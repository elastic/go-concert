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
	"github.com/stretchr/testify/require"
)

func TestSafeWaitGroup(t *testing.T) {
	t.Run("empty group", func(t *testing.T) {
		t.Run("wait returns", func(t *testing.T) {
			var wg SafeWaitGroup
			wg.Wait()
		})
		t.Run("safe to call wait multiple times", func(t *testing.T) {
			var wg SafeWaitGroup
			wg.Wait()
			wg.Wait()
			wg.Wait()
		})
		t.Run("safe to call wait after close", func(t *testing.T) {
			var wg SafeWaitGroup
			wg.Close()
			wg.Wait()
		})
		t.Run("fail to start after close", func(t *testing.T) {
			var wg SafeWaitGroup
			wg.Close()
			assert.Equal(t, ErrGroupClosed, wg.Add(1))
		})
		t.Run("fail to start after wait", func(t *testing.T) {
			var wg SafeWaitGroup
			wg.Wait()
			assert.Equal(t, ErrGroupClosed, wg.Add(1))
		})
	})

	// Test case reference see sync.WaitGroup at https://golang.org/src/sync/waitgroup_test.go
	t.Run("stress", func(t *testing.T) {
		n := 16
		var wg1, wg2 SafeWaitGroup
		exited := make(chan bool, n)

		wg1.Add(n)
		wg2.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				wg1.Done()
				wg2.Wait()
				exited <- true
			}()
		}
		wg1.Wait()

		for i := 0; i < n; i++ {
			select {
			case <-exited:
				t.Fatal("SafeWaitGroup released group too soon")
			default:
			}
			wg2.Done()
		}

		for i := 0; i != n; i++ {
			<-exited // Will block if barrier fails to unlock someone.
		}
	})

	t.Run("add and fail after Close", func(t *testing.T) {
		var wg SafeWaitGroup
		require.NoError(t, wg.Add(1))
		wg.Close()
		require.Equal(t, ErrGroupClosed, wg.Add(1))
	})

	t.Run("add and fail after Wait", func(t *testing.T) {
		var wg SafeWaitGroup
		require.NoError(t, wg.Add(1))
		wg.Done()
		wg.Wait()
		require.Equal(t, ErrGroupClosed, wg.Add(1))
	})

	t.Run("add negative delta releases resources", func(t *testing.T) {
		var wg SafeWaitGroup
		require.NoError(t, wg.Add(1))
		require.NoError(t, wg.Add(-1))
		wg.Wait() // will block if counter resource has not been released
	})

	t.Run("with context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		wg := SafeWaitGroupWithCancel(ctx)

		err := wg.Add(1)
		require.NoError(t, err, "expact waitgroup to be active")

		cancel()

		// cancel is evaluate async. Let's wait a little in case of a race or slow test environment
		start := time.Now()
		for {
			if time.Since(start) > 10*time.Second {
				t.Fatalf("SafeWaitGroup was not stopped")
			}

			err := wg.Add(1)
			if err == ErrGroupClosed {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}
