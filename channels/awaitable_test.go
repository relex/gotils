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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const waitDuration = 10 * time.Millisecond

// TestAwaitableBaseAndSignal tests AwaitableBase functions by SignalAwaitable
func TestAwaitableBaseAndSignal(t *testing.T) {
	s := NewSignalAwaitable()
	assert.False(t, s.Wait(waitDuration), ".Wait() should fail after timeout")
	assert.False(t, s.Peek(), ".Peek() should fail before signaling")
	s.Signal()
	assert.True(t, s.Wait(waitDuration), ".Wait() should succeed after signaling")
	assert.True(t, s.Peek(), ".Peek() should succeed after signaling")
	_, ok := <-s.Channel()
	assert.False(t, ok, "channel should be closed after signaling")
}

// TestAwaitableAfter tests AwaitableBase.After chained actions
func TestAwaitableAfter(t *testing.T) {
	s := NewSignalAwaitable()
	s1 := s.After(waitDuration)
	s.Signal()
	assert.False(t, s1.Peek(), ".Peek() of chained signal should fail right after signaling")
	assert.True(t, s.Wait(waitDuration), ".Wait() should succeed after signaling")
	assert.True(t, s1.Wait(2*waitDuration), ".Wait() of chain signal #1 should succeed after signaling")
}

// TestAwaitableNext tests AwaitableBase.Next chained actions
func TestAwaitableNext(t *testing.T) {
	states := make([]bool, 3)
	s := NewSignalAwaitable()
	s1 := s.Next(func() { states[0] = true })
	s2 := s.Next(func() { states[1] = true })
	s3 := s1.Next(func() { states[2] = true })
	assert.False(t, s3.Peek(), ".Peek() of chained signal should fail before signaling")
	s.Signal()
	assert.True(t, s.Wait(waitDuration), ".Wait() should succeed after signaling")
	assert.True(t, s1.Wait(waitDuration), ".Wait() of chain signal #1 should succeed after signaling")
	assert.True(t, s2.Wait(waitDuration), ".Wait() of chain signal #2 should succeed after signaling")
	assert.True(t, s3.Wait(waitDuration), ".Wait() of chain signal #3 should succeed after signaling")
	assert.True(t, states[0], "chain action #1 should be triggered after signaling")
	assert.True(t, states[1], "chain action #2 should be triggered after signaling")
	assert.True(t, states[2], "chain action #3 should be triggered after signaling")
}

// TestAllAwaitables tests AllAwaitables
func TestAllAwaitables(t *testing.T) {
	s1 := NewSignalAwaitable()
	s2 := NewSignalAwaitable()
	s3 := NewSignalAwaitable()
	sall := AllAwaitables(s1, s2, s3)
	assert.False(t, sall.Wait(waitDuration), ".Wait() should fail initially")
	s1.Signal()
	assert.False(t, sall.Wait(waitDuration), ".Wait() should fail if only some of awaitables are signaled")
	s2.Signal()
	assert.False(t, sall.Wait(waitDuration), ".Wait() should fail if only some of awaitables are signaled")
	s3.Signal()
	assert.True(t, sall.Wait(waitDuration), ".Wait() should succeed after all of awaitables are signaled")
}

// TestAnyAwaitables tests AnyAwaitables
func TestAnyAwaitables(t *testing.T) {
	s1 := NewSignalAwaitable()
	s2 := NewSignalAwaitable()
	s3 := NewSignalAwaitable()
	sany := AnyAwaitables(s1, s2, s3)
	assert.False(t, sany.Wait(waitDuration), ".Wait() should fail initially")
	s2.Signal()
	assert.True(t, sany.Wait(waitDuration), ".Wait() should succeed after one of awaitables are signaled")
}

// TestRemoveItemFromSlice tests removeSelectCaseByIndex
func TestRemoveItemFromSlice(t *testing.T) {
	c0 := reflect.SelectCase{Dir: reflect.SelectDir(0)}
	c1 := reflect.SelectCase{Dir: reflect.SelectDir(1)}
	c2 := reflect.SelectCase{Dir: reflect.SelectDir(2)}
	c3 := reflect.SelectCase{Dir: reflect.SelectDir(3)}
	slice := []reflect.SelectCase{c0, c1, c2, c3}
	assert.ElementsMatch(t, removeSelectCaseByIndex(slice, 0), []reflect.SelectCase{c1, c2, c3}, "contains right elements after removing [0]")
	assert.ElementsMatch(t, removeSelectCaseByIndex(slice, 1), []reflect.SelectCase{c0, c2, c3}, "contains right elements after removing [1]")
	assert.ElementsMatch(t, removeSelectCaseByIndex(slice, 2), []reflect.SelectCase{c0, c1, c3}, "contains right elements after removing [2]")
	assert.ElementsMatch(t, removeSelectCaseByIndex(slice, 3), []reflect.SelectCase{c0, c1, c2}, "contains right elements after removing [3]")
	assert.ElementsMatch(t, removeSelectCaseByIndex(removeSelectCaseByIndex(removeSelectCaseByIndex(slice, 3), 1), 1),
		[]reflect.SelectCase{c0}, "contains right elements after removing [*]")
}

// TestWaitGroupAwaitable tests waitGroupAwaitable
func TestWaitGroupAwaitable(t *testing.T) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	waitGroup.Add(1)
	waitGroup.Add(1)
	waitGroup.Done()
	a := NewWaitGroupAwaitable(waitGroup)
	assert.False(t, a.Wait(waitDuration), ".Wait() should fail if counter is not zero")
	waitGroup.Done()
	waitGroup.Done()
	assert.True(t, a.Wait(waitDuration), ".Wait() should succeed if counter is zero")
	waitGroup.Add(1)
	assert.True(t, a.Wait(waitDuration), ".Wait() should remain successful even if counter is increased again from zero")
}
