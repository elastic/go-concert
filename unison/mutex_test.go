// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package unison

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMutex(t *testing.T) {
	t.Run("zero value", testMutexZeroValue)
	t.Run("initialized", testInitializedMutex)
}

func testMutexZeroValue(t *testing.T) {
	zeroMutex := func() (m Mutex) { return m }
	testLockedFails(t, zeroMutex)
	testUnlockedFails(t, zeroMutex)

	t.Run("lock timeout -1 fails", func(t *testing.T) {
		var m Mutex
		assert.Equal(t, false, m.LockTimeout(-1))
	})
}

func testInitializedMutex(t *testing.T) {
	lockedMutex := func() Mutex {
		m := MakeMutex()
		m.Lock()
		return m
	}
	unlockedMutex := MakeMutex

	testUnlockedFails(t, unlockedMutex)
	testLockedFails(t, lockedMutex)

	t.Run("lock unlocked with timeout -1 succeeds", func(t *testing.T) {
		m := MakeMutex()
		assert.Equal(t, true, m.LockTimeout(-1))
	})

	t.Run("lock unlocked with timeout 0 succeeds", func(t *testing.T) {
		m := MakeMutex()
		assert.Equal(t, true, m.LockTimeout(0))
	})

	t.Run("lock unlocked with large timeout succeeds", func(t *testing.T) {
		m := MakeMutex()
		assert.Equal(t, true, m.LockTimeout(10*time.Minute))
	})

	t.Run("try lock on unlocked mutex succeeds", func(t *testing.T) {
		m := MakeMutex()
		assert.Equal(t, true, m.TryLock())
	})

	t.Run("lock unlocked with context succeeds", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go cancel()
		m := MakeMutex()
		assert.Equal(t, nil, m.LockContext(ctx))
	})
}

func testLockedFails(t *testing.T, create func() Mutex) {
	t.Run("lock timeout 0 fails", func(t *testing.T) {
		var m Mutex
		assert.Equal(t, false, m.LockTimeout(1))
	})

	t.Run("lock with timeout fails", func(t *testing.T) {
		m := create()
		assert.Equal(t, false, m.LockTimeout(10*time.Millisecond))
	})

	t.Run("trylock fails", func(t *testing.T) {
		m := create()
		assert.Equal(t, false, m.TryLock())
	})

	t.Run("lock with context canceling", func(t *testing.T) {
		m := create()
		ctx, cancel := context.WithCancel(context.Background())
		go cancel()
		assert.Equal(t, context.Canceled, m.LockContext(ctx))
	})

	t.Run("lock with already cancalled context", func(t *testing.T) {
		m := create()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Equal(t, context.Canceled, m.LockContext(ctx))
	})
}

func testUnlockedFails(t *testing.T, create func() Mutex) {
	t.Run("unlock on unlocked panics", func(t *testing.T) {
		expectPanic(t, func() {
			m := create()
			m.Unlock()
		})
	})
}

func expectPanic(t *testing.T, fn func()) {
	defer func() {
		if x := recover(); x == nil {
			t.Fatal("did expect the call to Panic")
		}
	}()
	fn()
}
