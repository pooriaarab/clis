// Package backoff retries a check with a doubling delay.
package backoff

import (
	"context"
	"errors"
	"time"
)

// ErrTimeout means the check was still not done when the wait ended.
var ErrTimeout = errors.New("not ready before the wait ended")

// Max caps the delay between two tries.
const Max = 60 * time.Second

// Opts controls one wait.
type Opts struct {
	Wait     time.Duration // total time before giving up
	Interval time.Duration // first delay; it doubles up to Max
	// OnRetry is called before each wait. It may be nil.
	OnRetry func(attempt int, delay time.Duration)
}

// Until calls try until it reports done, returns an error, or the wait ends.
// It returns the number of tries.
func Until(ctx context.Context, o Opts, try func() (done bool, err error)) (int, error) {
	deadline := time.Now().Add(o.Wait)
	delay := o.Interval
	for attempt := 1; ; attempt++ {
		done, err := try()
		if done || err != nil {
			return attempt, err
		}
		if time.Now().Add(delay).After(deadline) {
			return attempt, ErrTimeout
		}
		if o.OnRetry != nil {
			o.OnRetry(attempt, delay)
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return attempt, ctx.Err()
		}
		delay = min(delay*2, Max)
	}
}
