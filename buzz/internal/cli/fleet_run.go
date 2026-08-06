package cli

// fleet --run supervisor: launches one buzz-acp child per agent created by
// `fleet`, reusing the same env-building + exec.CommandContext spawn logic
// as `agents run` (buildAgentRuntimeEnv in cli.go), but supervising N
// children instead of one: a --max-concurrent cap (excess agents queue
// behind it), per-agent log files, crash+backoff restart, and graceful
// shutdown on SIGINT/SIGTERM.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// fleetAgentStatus is the per-agent row printed in the fleet manifest and
// the final summary.
type fleetAgentStatus struct {
	Name     string `json:"name"`
	PubKey   string `json:"pubkey"`
	LogPath  string `json:"log_path"`
	PID      int    `json:"pid"`
	Restarts int    `json:"restarts"`
	State    string `json:"state"`
}

const (
	fleetInitialBackoff = time.Second
	fleetMaxBackoff     = 30 * time.Second
	// fleetShutdownGrace is how long a child gets to exit after SIGTERM
	// (via exec.Cmd.Cancel/WaitDelay) before it is killed on shutdown.
	fleetShutdownGrace = 10 * time.Second
)

// runFleet launches and supervises one buzz-acp child per created agent.
// It blocks: it prints the fleet manifest (one JSON document) once every
// agent has made its first start attempt, then supervises until a shutdown
// signal, then prints a final summary (a second JSON document).
func (opts *rootOptions) runFleet(ctx context.Context, agents []createdAgent, maxConcurrent int, logDir, acpCommand, harnessCommand string) error {
	if strings.TrimSpace(acpCommand) == "" {
		return inputError("acp-command is required")
	}
	resolved, err := resolveNoKeys(opts)
	if err != nil {
		return err
	}
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), fmt.Sprintf("buzz-fleet-%d", time.Now().UnixNano()))
	}
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return otherWrap("create fleet log directory", err)
	}

	runCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	statuses := make([]fleetAgentStatus, len(agents))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)
	started := make(chan struct{}, len(agents))

	for i, agent := range agents {
		logPath := filepath.Join(logDir, agent.Name+".log")
		statuses[i] = fleetAgentStatus{Name: agent.Name, PubKey: agent.PubKey, LogPath: logPath, State: "queued"}
		wg.Add(1)
		go func(i int, agent createdAgent) {
			defer wg.Done()
			supervise(runCtx, agent, resolved.RelayURL, acpCommand, harnessCommand, statuses[i].LogPath, sem, &mu, &statuses[i], started)
		}(i, agent)
	}

waitStarted:
	for i := 0; i < len(agents); i++ {
		select {
		case <-started:
		case <-runCtx.Done():
			break waitStarted
		}
	}
	mu.Lock()
	manifest := append([]fleetAgentStatus{}, statuses...)
	mu.Unlock()
	if err := opts.writeJSON(map[string]any{"manifest": manifest, "log_dir": logDir}); err != nil {
		return err
	}

	wg.Wait()
	mu.Lock()
	summary := append([]fleetAgentStatus{}, statuses...)
	mu.Unlock()
	return opts.writeJSON(map[string]any{"summary": summary})
}

// supervise runs one agent's buzz-acp child in a restart loop with
// exponential backoff, honoring maxConcurrent via sem, and stopping
// gracefully when ctx is canceled. It reports its first start attempt
// (success or failure) on started exactly once, so the caller can print a
// manifest without waiting for every crash-restart cycle.
func supervise(ctx context.Context, agent createdAgent, relayURL, acpCommand, harnessCommand, logPath string, sem chan struct{}, mu *sync.Mutex, status *fleetAgentStatus, started chan struct{}) {
	reportStart := func() {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	backoff := fleetInitialBackoff
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			setState(mu, status, "stopped", 0)
			if attempt == 0 {
				reportStart()
			}
			return
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			setState(mu, status, "stopped", 0)
			if attempt == 0 {
				reportStart()
			}
			return
		}

		logFile, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if openErr != nil {
			<-sem
			setState(mu, status, "error: "+openErr.Error(), 0)
			if attempt == 0 {
				reportStart()
			}
			return
		}

		proc := exec.CommandContext(ctx, acpCommand)
		proc.Env = buildAgentRuntimeEnv(agent.Nsec, relayURL, agent.AuthTag, harnessCommand)
		proc.Stdout = logFile
		proc.Stderr = logFile
		// Graceful shutdown: on ctx cancellation exec would otherwise kill
		// the child immediately. Send SIGTERM instead and give it
		// fleetShutdownGrace before the stdlib escalates to a hard kill.
		proc.Cancel = func() error { return proc.Process.Signal(syscall.SIGTERM) }
		proc.WaitDelay = fleetShutdownGrace

		startErr := proc.Start()
		if startErr != nil {
			logFile.Close()
			<-sem
			setState(mu, status, "error: "+startErr.Error(), 0)
			if attempt == 0 {
				reportStart()
			}
			return
		}
		setState(mu, status, "running", proc.Process.Pid)
		if attempt == 0 {
			reportStart()
		}

		waitErr := proc.Wait()
		logFile.Close()
		<-sem

		if ctx.Err() != nil {
			setState(mu, status, "stopped", 0)
			return
		}

		mu.Lock()
		status.Restarts++
		status.PID = 0
		if waitErr != nil {
			status.State = "crashed"
		} else {
			status.State = "exited"
		}
		mu.Unlock()

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			setState(mu, status, "stopped", 0)
			return
		}
		if backoff < fleetMaxBackoff {
			backoff *= 2
			if backoff > fleetMaxBackoff {
				backoff = fleetMaxBackoff
			}
		}
	}
}

func setState(mu *sync.Mutex, status *fleetAgentStatus, state string, pid int) {
	mu.Lock()
	status.State = state
	status.PID = pid
	mu.Unlock()
}
