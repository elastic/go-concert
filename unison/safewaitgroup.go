package unison

import (
	"errors"
	"sync"
)

// SafeWaitGroup provides a safe alternative to WaitGroup, that instead of
// panicing returns an error when Wait has been called.
type SafeWaitGroup struct {
	mu     sync.RWMutex
	wg     sync.WaitGroup
	closed bool
}

// ErrGroupClosed indicates that the WaitGroup is currently closed, and no more
// routines can be started.
var ErrGroupClosed = errors.New("group closed")

// Add adds the delta to the WaitGroup counter.
// If the counter becomes 0, all goroutines are blocked on Wait will continue.
//
// Add returns an error if 'Wait' has already been called, indicating that no more
// go-routines should be started.
func (s *SafeWaitGroup) Add(n int) error {
	if n < 0 {
		s.wg.Add(n)
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrGroupClosed
	}

	s.wg.Add(n)
	return nil
}

// Done decrements the WaitGroup counter.
func (s *SafeWaitGroup) Done() {
	s.wg.Done()
}

// Close marks the wait group as closed. All calls to Add will fail with ErrGroupClosed after
// close has been called. Close does not wait until the WaitGroup counter has
// reached zero, but will return immediately. Use Wait to wait for the counter to become 0.
func (s *SafeWaitGroup) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// Wait closes the WaitGroup and blocks until the WaitGroup counter is zero.
// Add will return errors the moment 'Wait' has been called.
func (s *SafeWaitGroup) Wait() {
	s.Close()
	s.wg.Wait()
}
