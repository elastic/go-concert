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
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockManager(t *testing.T) {
	// test some internals, just to make sure the ManagedLock correctly
	// frees/deletes resources not needed anymore
	t.Run("internals", func(t *testing.T) {
		t.Run("shared lock entries are allocated lazily", func(t *testing.T) {
			lm := NewLockManager()
			lock := lm.Access("key")

			require.Equal(t, 0, len(lm.table), "expected to have no shared entry in the table")

			lock.Lock()
			require.Equal(t, 1, len(lm.table), "expected shared entry to be generated when locking")

			lock.Unlock()
			require.Equal(t, 0, len(lm.table), "expected shared entry to be freed after unlock")
		})

		t.Run("garbage collecting a ManagedLock frees shared entries in the LockManager", func(t *testing.T) {
			lm := NewLockManager()

			func() {
				lock := lm.Access("key")
				lock.Lock()
				require.Equal(t, 1, len(lm.table), "expected shared entry to be generated when locking")
			}()

			for i := 0; i < 10; i++ {
				runtime.GC()
			}
			require.Equal(t, 0, len(lm.table), "expected shared entry to be freed after unlock")
		})
	})

	t.Run("panic if we attempt to lock the same instance twice", func(t *testing.T) {
		lm := NewLockManager()
		lock := lm.Access("key")
		lock.Lock()
		defer lock.Unlock()
		expectPanic(t, func() { lock.Lock() })
	})

	t.Run("mutex can only be held by one lock instance", func(t *testing.T) {
		lm := NewLockManager()
		lock1, lock2 := lm.Access("key"), lm.Access("key")

		_, locked := lock1.TryLock()
		require.True(t, locked)

		_, locked = lock2.TryLock()
		require.False(t, locked)

		lock1.Unlock()
	})

	t.Run("mutex is transmissible", func(t *testing.T) {
		lm := NewLockManager()
		lock := lm.Access("key")
		lock.Lock()

		// create second go-routine that will block unless it can acquire the lock
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := lm.Access("key")
			lock.Lock()
			lock.Unlock()
		}()

		// unlock and wait for helper go-routine to quit
		lock.Unlock()
		wg.Wait() // <- deadlock if the test failed
	})

	t.Run("garbage collector releases lock", func(t *testing.T) {
		lm := NewLockManager()
		lock := lm.Access("key")

		finalizerCalled := false
		oldFinalizer := managedLockFinalizer
		managedLockFinalizer = func(l *ManagedLock) {
			finalizerCalled = true
			oldFinalizer(l)
		}
		defer func() { managedLockFinalizer = oldFinalizer }()

		func() {
			tmp := lm.Access("key")
			tmp.Lock()
		}()

		for i := 0; i < 10; i++ {
			runtime.GC()
		}
		_, locked := lock.TryLock()
		require.True(t, locked)
		lock.Unlock()

		assert.True(t, finalizerCalled)
	})

	t.Run("lock with timeout", func(t *testing.T) {
		t.Run("success if not taken", func(t *testing.T) {
			lm := NewLockManager()
			lock := lm.Access("key")
			_, locked := lock.LockTimeout(10 * time.Millisecond)
			require.True(t, locked)
		})

		t.Run("success if freed before timeout ends", func(t *testing.T) {
			lm := NewLockManager()
			lock := lm.Access("key")
			lock.Lock()

			var wg, wgStart sync.WaitGroup
			wg.Add(1)
			wgStart.Add(1)
			go func() {
				defer wg.Done()
				lock := lm.Access("key")
				wgStart.Done()
				_, locked := lock.LockTimeout(1 * time.Second)
				if !locked {
					panic("expected to get the lock")
				}
				lock.Unlock()
			}()

			wgStart.Wait()
			time.Sleep(50 * time.Microsecond)
			lock.Unlock()
			wg.Wait() // <- deadlock if LockTimeout did not succeed
		})

		t.Run("fail if taken and no timeout", func(t *testing.T) {
			lm := NewLockManager()
			l1, l2 := lm.Access("key"), lm.Access("key")
			l1.Lock()
			defer l1.Unlock()

			_, locked := l2.LockTimeout(0)
			assert.False(t, locked)
		})

		t.Run("fail after timeout", func(t *testing.T) {
			lm := NewLockManager()
			l1, l2 := lm.Access("key"), lm.Access("key")
			l1.Lock()
			defer l1.Unlock()

			_, locked := l2.LockTimeout(50 * time.Millisecond)
			assert.False(t, locked)
		})
	})

	t.Run("lock with context", func(t *testing.T) {
		t.Run("success if context is not cancelled", func(t *testing.T) {
			lm := NewLockManager()
			l := lm.Access("key")
			_, err := l.LockContext(context.Background())
			require.NoError(t, err)
			l.Unlock()
		})

		t.Run("fail if context was already cancelled", func(t *testing.T) {
			lm := NewLockManager()
			l := lm.Access("key")
			ctx, cancelFn := context.WithCancel(context.Background())
			cancelFn()
			_, err := l.LockContext(ctx)
			require.Equal(t, ctx.Err(), err)
		})

		t.Run("fail if context gets cancelled while we are waiting", func(t *testing.T) {
			lm := NewLockManager()
			l1, l2 := lm.Access("key"), lm.Access("key")

			l1.Lock()
			defer l1.Unlock()

			ctx, cancelFn := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancelFn()
			_, err := l2.LockContext(ctx)
			assert.Equal(t, ctx.Err(), err)
		})
	})

	t.Run("still need to unlock after forced lock release", func(t *testing.T) {
		t.Run("panics if we attempt to lock right away", func(t *testing.T) {
			lm := NewLockManager()
			l := lm.Access("key")
			var sigDone, sigLost, sigUnlocked int
			session := l.Lock(WithSignalCallbacks{
				Done:     func() { sigDone++ },
				Lost:     func() { sigLost++ },
				Unlocked: func() { sigUnlocked++ },
			})
			lm.ForceUnlock("key")

			<-session.LockLost() // wait for signal before trying to lock again
			expectPanic(t, func() {
				l.Lock()
			})

			assert.Equal(t, 1, sigDone)
			assert.Equal(t, 1, sigLost)
			assert.Equal(t, 0, sigUnlocked)
		})

		t.Run("unlock does succeed", func(t *testing.T) {
			lm := NewLockManager()
			l := lm.Access("key")
			session := l.Lock()
			lm.ForceUnlock("key")

			<-session.LockLost() // wait for signal before trying to lock again
			l.Unlock()
		})
	})

	t.Run("check ForceUnlockAll", func(t *testing.T) {
		lm := NewLockManager()
		l1, l2 := lm.Access("key1"), lm.Access("key2")
		s1, s2 := l1.Lock(), l2.Lock()
		defer l1.Unlock()
		defer l2.Unlock()
		lm.ForceUnlockAll()
		<-s1.LockLost()
		<-s2.LockLost()
	})

	t.Run("signaling", func(t *testing.T) {
		const sigUnlocked = 1
		const sigForced = 2

		waitDone := func(session *LockSession, fn func()) {
			<-session.Done()
			fn()
		}

		waitSignal := func(resp chan int, session *LockSession) {
			select {
			case <-session.Unlocked():
				resp <- sigUnlocked
			case <-session.LockLost():
				resp <- sigForced
			}
		}

		t.Run("lock session reports done on unlock", func(t *testing.T) {
			lm := NewLockManager()
			lock := lm.Access("key")
			var sigDone, sigUnlocked int
			session := lock.Lock(WithSignalCallbacks{
				Done:     func() { sigDone++ },
				Unlocked: func() { sigUnlocked++ },
			})

			var wg sync.WaitGroup
			wg.Add(1)
			go waitDone(session, wg.Done)

			lock.Unlock()
			wg.Wait() // <- deadlock if signal is not send to session

			assert.Equal(t, 1, sigDone)
			assert.Equal(t, 1, sigUnlocked)
		})

		t.Run("lock session reports 'unlocked'", func(t *testing.T) {
			lm := NewLockManager()
			lock := lm.Access("key")
			session := lock.Lock()

			resp := make(chan int)
			go waitSignal(resp, session)

			lock.Unlock()
			assert.Equal(t, sigUnlocked, <-resp)
		})
	})
}
