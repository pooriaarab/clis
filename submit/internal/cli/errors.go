package cli

import (
	"errors"
	"fmt"
)

// Exit codes. Scripts can tell a usage mistake from a real failure.
const (
	ExitFailure = 1
	ExitUsage   = 2
)

// ExitError carries the process exit code with the message.
type ExitError struct {
	Code int
	Err  error
	// Printed means the command already showed the failure in its own output.
	Printed bool
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// exitCode returns the code for err. Errors that did not come from a command
// body are cobra usage errors: unknown command, bad flag, wrong argument count.
func exitCode(err error) int {
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitUsage
}

// run wraps a command body so a plain error exits 1.
func run(fn func(args []string) error) func(*cobraCmd, []string) error {
	return func(_ *cobraCmd, args []string) error {
		err := fn(args)
		var ee *ExitError
		if err == nil || errors.As(err, &ee) {
			return err
		}
		return &ExitError{Code: ExitFailure, Err: err}
	}
}

// usagef builds a usage error (exit 2): bad flags, bad files, unknown names.
func usagef(format string, a ...any) error {
	return &ExitError{Code: ExitUsage, Err: fmt.Errorf(format, a...)}
}
