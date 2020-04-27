package unison

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
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
}
