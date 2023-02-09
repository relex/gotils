package clock

import (
	"time"
)

// SafeTimer wraps the Go timer to provide deadlock-free resetting
type SafeTimer struct {
	timer   *time.Timer
	timeout time.Duration
}

func NewSafeTimer(timeout time.Duration) *SafeTimer {
	return &SafeTimer{
		timer:   time.NewTimer(timeout),
		timeout: timeout,
	}
}

func (stimer *SafeTimer) C() <-chan time.Time {
	return stimer.timer.C
}

func (stimer *SafeTimer) Reset() {
	stimer.Stop()
	stimer.timer.Reset(stimer.timeout)
}

func (stimer *SafeTimer) Stop() {
	// Correct timer stopping from https://github.com/golang/go/issues/27169
	if !stimer.timer.Stop() {
		select {
		case <-stimer.timer.C:
		default:
		}
	}
}
