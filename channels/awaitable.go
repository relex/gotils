// Copyright 2021 RELEX Oy
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package channels

import (
	"reflect"
	"sync"
	"time"
)

// Awaitable is a signal that can waited on.
type Awaitable interface {
	After(timeout time.Duration) Awaitable
	Channel() <-chan Void
	Next(action func()) Awaitable
	Peek() bool
	Wait(timeout time.Duration) bool
	WaitForever()
	WaitTimer(timerC <-chan time.Time) bool
}

// AwaitableBase provides waiting methods by a channel (to be closed)
type AwaitableBase struct {
	channel chan Void
}

func newAwaitableBase() AwaitableBase {
	return AwaitableBase{
		channel: make(chan Void),
	}
}

// Channel returns the internal channel that can be used in more complex situations.
// The returned channel is meant to be read-only and select should either be pending or not-ok,
// because the signal is done by closing the channel, not by sending any message.
func (awaitable *AwaitableBase) Channel() <-chan Void {
	return awaitable.channel
}

// After creates an awaitable which is signaled after this awaitable and certain duration of time
// It returns a chained Awaitable; The current Awaitable can still be waited on.
func (awaitable AwaitableBase) After(timeout time.Duration) Awaitable {
	nextSignal := NewSignalAwaitable()
	go func() {
		awaitable.WaitForever()
		time.Sleep(timeout)
		nextSignal.Signal()
	}()
	return nextSignal
}

// Next chains an action to be executed when the current Awaitable is done/signaled (no timeout)
// It returns a chained Awaitable; The current Awaitable can still be waited on.
func (awaitable AwaitableBase) Next(action func()) Awaitable {
	nextSignal := NewSignalAwaitable()
	go func() {
		awaitable.WaitForever()
		action()
		nextSignal.Signal()
	}()
	return nextSignal
}

// Peek returns true if the signal has come. It doesn't wait.
func (awaitable *AwaitableBase) Peek() bool {
	select {
	case <-awaitable.channel:
		return true
	default:
		return false
	}
}

// Wait waits for the signal until specified timeout.
// Returns true if sucessful or false if timeout
func (awaitable *AwaitableBase) Wait(timeout time.Duration) bool {
	select {
	case <-awaitable.channel:
		return true
	case <-time.After(timeout):
		return false
	}
}

// WaitForever waits for the signal
func (awaitable *AwaitableBase) WaitForever() {
	<-awaitable.channel
}

// WaitTimer waits for the signal until the timer is triggered (by time/timer.C)
// Returns true if sucessful or false if timer is triggered
func (awaitable *AwaitableBase) WaitTimer(timerC <-chan time.Time) bool {
	select {
	case <-awaitable.channel:
		return true
	case <-timerC:
		return false
	}
}

// SignalAwaitable is a one-time signal that can waited on.
// It's implemented by a simple channel without any message
type SignalAwaitable struct {
	AwaitableBase
}

// NewSignalAwaitable creates a SignalAwaitable / one-time signal to be waited on.
func NewSignalAwaitable() *SignalAwaitable {
	return &SignalAwaitable{
		newAwaitableBase(),
	}
}

// Signal marks the Awaitable to notify the awaiter(s)
// It can be only called once or panic
func (awaitable *SignalAwaitable) Signal() {
	close(awaitable.channel)
}

// AllAwaitables creates an aggregated Awaitable waiting for all of the given Awaitable(s)
func AllAwaitables(awaitables ...Awaitable) Awaitable {
	aggregated := NewSignalAwaitable()
	caseList := make([]reflect.SelectCase, len(awaitables))
	for index, a := range awaitables {
		caseList[index] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(a.Channel()),
		}
	}
	go func() {
		remainingCases := caseList
		for len(remainingCases) > 0 {
			index, _, _ := reflect.Select(remainingCases)
			remainingCases = removeSelectCaseByIndex(remainingCases, index)
		}
		aggregated.Signal()
	}()
	return aggregated
}

// AnyAwaitables creates an aggregated Awaitable waiting for any of the given Awaitable(s)
func AnyAwaitables(awaitables ...Awaitable) Awaitable {
	aggregated := NewSignalAwaitable()
	caseList := make([]reflect.SelectCase, len(awaitables))
	for index, a := range awaitables {
		caseList[index] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(a.Channel()),
		}
	}
	go func() {
		reflect.Select(caseList)
		aggregated.Signal()
	}()
	return aggregated
}

type waitGroupAwaitable struct {
	AwaitableBase
	waitGroup *sync.WaitGroup
}

// NewWaitGroupAwaitable creates an Awaitable waiting on the given sync.WaitGroup
// The waiting on the given WaitGroup starts immediately in this call and ends when the counter reaches zero.
// Subsequent changing of the counter would have no effect - a new Awaitable would need to be created.
func NewWaitGroupAwaitable(waitGroup *sync.WaitGroup) Awaitable {
	awaitable := &waitGroupAwaitable{
		AwaitableBase: newAwaitableBase(),
		waitGroup:     waitGroup,
	}
	go func() {
		waitGroup.Wait()
		close(awaitable.channel)
	}()
	return awaitable
}

func removeSelectCaseByIndex(slice []reflect.SelectCase, index int) []reflect.SelectCase {
	if index == 0 {
		return slice[1:]
	}
	if index == len(slice)-1 {
		return slice[:index]
	}
	newSlice := make([]reflect.SelectCase, len(slice)-1)
	copy(newSlice, slice[:index])
	copy(newSlice[index:], slice[index+1:])
	return newSlice
}
