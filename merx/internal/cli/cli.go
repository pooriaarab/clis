package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"merx-cli/internal/httpx"
)

const portal = "https://www.merx.com/"

var (
	flagCache string
	flagJSON  bool
)

func Execute() error { return root().Execute() }

func root() *cobra.Command {
	cmd := &cobra.Command{Use: "merx", Short: "Fetch and cache MERX procurement notices", SilenceUsage: true, SilenceErrors: true}
	cmd.PersistentFlags().StringVar(&flagCache, "cache-dir", "", "cache directory")
	cmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "JSON on stdout")
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.AddCommand(doctorCmd(), searchCmd(), showCmd())
	return cmd
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Report cache configuration and reachability", RunE: func(*cobra.Command, []string) error {
		dir, err := cacheDir()
		if err != nil {
			return err
		}
		used, err := diskUsed(dir)
		if err != nil {
			return err
		}
		c, err := httpx.New()
		if err != nil {
			return err
		}
		ok, pingErr := ping(c)
		if flagJSON {
			out := map[string]any{"ok": ok, "cache_dir": dir, "bytes": used, "reachable": ok, "host": portal}
			if pingErr != nil {
				out["error"] = pingErr.Error()
			}
			return emit(out)
		}
		if pingErr != nil {
			fmt.Printf("cache: %s\ndisk: %d bytes\nreachable: %t (%v)\n", dir, used, ok, pingErr)
		} else {
			fmt.Printf("cache: %s\ndisk: %d bytes\nreachable: %t\n", dir, used, ok)
		}
		return nil
	}}
}

// cacheDir: --cache-dir, then MERX_CACHE_DIR, then XDG, then ~/.cache.
func cacheDir() (string, error) {
	if flagCache != "" {
		return flagCache, nil
	}
	if e := os.Getenv("MERX_CACHE_DIR"); e != "" {
		return e, nil
	}
	if e := os.Getenv("XDG_CACHE_HOME"); e != "" {
		return filepath.Join(e, "merx"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "merx"), nil
}

func diskUsed(dir string) (int64, error) {
	var n int64
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		n += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return n, err
}

func ping(c *httpx.Client) (bool, error) {
	req, err := httpx.NewRequest("HEAD", portal)
	if err != nil {
		return false, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
