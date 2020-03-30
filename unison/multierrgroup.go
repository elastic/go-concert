package unison

import (
	"context"
	"sync"
)

// MultiErrGroup is a collection of goroutines working on subtasks
// concurrently.  The group waits until all subtasks have finished and collects
// all errors encountered.
//
// The zero value of MultiErrGroup is a valid group.
type MultiErrGroup struct {
	mu   sync.Mutex
	errs []error
	wg   sync.WaitGroup
}

// Go starts a new go-routine, collecting errors encounted into the
// MultiErrGroup.
func (g *MultiErrGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		err := fn()
		if err != nil && err != context.Canceled {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.errs = append(g.errs, err)
		}
	}()
}

// Wait waits until all go-routines have been stopped and returns all errors
// encountered.
func (g *MultiErrGroup) Wait() []error {
	g.wg.Wait()
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.errs
}
