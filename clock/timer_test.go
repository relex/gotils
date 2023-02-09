package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeTimerReset(t *testing.T) {
	timer := NewSafeTimer(1 * time.Millisecond)

	// fetch the time
	tm := <-timer.C()
	assert.NotEmpty(t, tm)

	select {
	case <-timer.C():
		t.Error("timer channel should be empty")
	case <-time.After(100 * time.Millisecond):
		t.Log("success reading from empty channel")
	}

	timer.Reset()
	select {
	case <-timer.C():
		t.Log("success reading new time")
	case <-time.After(100 * time.Millisecond):
		t.Error("timer channel should not be empty")
	}
}

func TestSafeTimerResetDraining(t *testing.T) {
	timer := NewSafeTimer(1 * time.Millisecond)

	// wait for timer to send time
	time.Sleep(100 * time.Millisecond)
	oldTime := time.Now()

	// the channel contains time now and should be drained in Reset()
	timer.Reset()

	select {
	case newTime := <-timer.C():
		assert.True(t, oldTime.Before(newTime))
	case <-time.After(100 * time.Millisecond):
		t.Error("timer channel should not be empty")
	}
}
