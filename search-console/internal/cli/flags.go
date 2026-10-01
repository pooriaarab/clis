package cli

import (
	"fmt"
	"time"
)

// needPositive rejects a duration that is zero or negative. It is a usage error.
func needPositive(flag string, d time.Duration) error {
	if d <= 0 {
		return &ExitError{Code: ExitUsage, Err: fmt.Errorf("%s must be greater than 0 (got %s)", flag, d)}
	}
	return nil
}

// needNotNegative rejects a negative duration. Zero is allowed.
func needNotNegative(flag string, d time.Duration) error {
	if d < 0 {
		return &ExitError{Code: ExitUsage, Err: fmt.Errorf("%s must not be negative (got %s)", flag, d)}
	}
	return nil
}

// checkPolling validates the --wait and --interval pair every polling command takes.
func checkPolling(wait, interval time.Duration) error {
	if err := needNotNegative("--wait", wait); err != nil {
		return err
	}
	return needPositive("--interval", interval)
}
