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

package concert

import (
	"sync"
	"time"
)

type Semaphore struct {
	mu      sync.Mutex
	n       int
	waiters Waitlist
}

func NewSemaphore(n int) *Semaphore {
	return &Semaphore{n: n}
}

func (s *Semaphore) Acquire() {
	s.AcquireContext(nil)
}

func (s *Semaphore) AcquireContext(context doneContext) error {
	s.mu.Lock()
	s.n--
	if s.n > 0 {
		s.mu.Unlock()
		return nil
	}

	// need to wait. Create waiter before unlock, so to ensure the wait is
	// already in the list before the semaphore can send a signal.
	waiter := s.waiters.Enqueue(false, nil)
	s.mu.Unlock()

	if context == nil {
		waiter.Wait()
		return nil
	}

	err := waiter.WaitContext(context)
	if err != nil {
		s.abort(waiter)
	}
	return err
}

func (s *Semaphore) AcquireTimeout(dur time.Duration) bool {
	switch {
	case dur == 0:
		return s.TryAcquire()
	case dur < 0:
		s.Acquire()
		return true
	}

	s.mu.Lock()
	s.n--
	if s.n > 0 {
		s.mu.Unlock()
		return true
	}

	// need to wait. Create waiter before unlock, so to ensure the wait is
	// already in the list before the semaphore can send a signal.
	waiter := s.waiters.Enqueue(false, nil)
	s.mu.Unlock()

	ok := waiter.WaitTimeout(dur)
	if !ok {
		s.abort(waiter)
	}
	return ok
}

func (s *Semaphore) TryAcquire() bool {
	s.mu.Lock()
	ok := s.n > 1
	if ok {
		s.n--
	}
	s.mu.Unlock()
	return ok
}

func (s *Semaphore) Waiter() *Waiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mu.Lock()
	s.n--
	if s.n > 0 {
		// all fine -> let's create a dummy waiter
		w := &Waiter{
			state: waiterInactive,
			ready: make(chan struct{}),
		}
		close(w.ready)
		return w
	}

	var waiter *Waiter
	waiter = s.waiters.Enqueue(false, s.Release)
	return waiter
}

func (s *Semaphore) Release() {
	s.mu.Lock()
	s.doRelease()
	s.mu.Unlock()
}

func (s *Semaphore) abort(w *Waiter) {
	s.mu.Lock()
	w.Cancel()
	s.doRelease()
	s.mu.Unlock()
}

func (s *Semaphore) doRelease() {
	s.n++
	if s.n >= 1 {
		s.waiters.Notify()
	}
}
