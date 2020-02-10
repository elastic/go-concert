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
	"sync"
	"time"
)

type Waitlist struct {
	mu         sync.Mutex
	head, tail *Waiter
}

type Waiter struct {
	list       *Waitlist
	next, prev *Waiter

	state waiterState
	ready chan struct{}

	propagateCancel bool
	onCancel        func()
}

type waiterState int

const (
	waiterActive waiterState = iota
	waiterNotified
	waiterBroadcasted
	waiterCancelled
	waiterInactive
)

func (l *Waitlist) Enqueue(propagateCancel bool, onCancel func()) *Waiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	w := &Waiter{
		list:            l,
		ready:           make(chan struct{}),
		onCancel:        onCancel,
		propagateCancel: propagateCancel,
	}

	if l.tail == nil {
		l.head, l.tail = w, w
		w.next, w.prev = w, w
	} else {
		w.prev, w.next = l.tail, l.head
		l.tail.next, l.head.prev = w, w
		l.tail = w
	}

	return w
}

func (l *Waitlist) Wait() {
	w := l.Enqueue(true, nil)
	w.Wait()
}

func (l *Waitlist) WaitContext(context doneContext) error {
	w := l.Enqueue(true, nil)
	return w.WaitContext(context)
}

func (l *Waitlist) Notify() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.notifyNext()
}

func (l *Waitlist) notifyNext() {
	if l.head == nil {
		return
	}

	w := l.head
	if w.next == w {
		l.head = nil
		l.tail = nil
	} else {
		w.next.prev = w.prev
		w.prev.next = w.next
		l.head = w.next
	}

	if w.state != waiterActive {
		panic("non active waiter in list")
	}

	w.state = waiterNotified
	close(w.ready)

}

func (l *Waitlist) Broadcast() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.head == nil {
		return
	}

	l.tail.next = nil
	for w := l.head; w != nil; w = w.next {
		if w.state != waiterActive {
			panic("non active waiter in list")
		}

		w.state = waiterBroadcasted
		close(w.ready)
	}
}

func (l *Waitlist) cancel(w *Waiter) {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch w.state {
	case waiterNotified:
		if w.propagateCancel {
			l.cancelNotified(w)
		} else {
			w.state = waiterCancelled
		}

	case waiterActive:
		l.cancelActive(w)
	}
}

// cancelNotified is used on a waiter to be cancelled, that already has been
// notified.  This can happen if the notification signal races with another
// signal in a channel.
// In case of a cancel, we continue propagating the 'Notify' signal to the next
// waiter, as it is assumed that the waiter has been removed.
// It is assumed that the lock to l.mu is held.
func (l *Waitlist) cancelNotified(w *Waiter) {
	l.notifyNext()
	w.state = waiterCancelled
}

// cancelActive removes the waiter from the list and cancels it.
// It is assumed that the lock to l.mu is held.
func (l *Waitlist) cancelActive(w *Waiter) {
	// remove from list
	if w.next == w {
		l.head = nil
		l.tail = nil
	} else {
		w.next.prev = w.prev
		w.prev.next = w.next
		if l.tail == w {
			l.tail = w.prev
		}
		if l.head == w {
			l.head = w.next
		}
	}

	w.state = waiterCancelled
	close(w.ready)
}

func (w *Waiter) C() <-chan struct{} {
	return w.ready
}

func (w *Waiter) Cancel() {
	if w.list != nil {
		w.list.cancel(w)
	}
	if w.onCancel != nil {
		w.onCancel()
	}
}

func (w *Waiter) Wait() {
	<-w.ready
}

func (w *Waiter) WaitContext(context doneContext) error {
	select {
	case <-w.ready:
		return nil
	case <-context.Done():
		return context.Err()
	}
}

func (w *Waiter) WaitTimeout(dur time.Duration) bool {
	timer := time.NewTimer(dur)
	select {
	case <-w.ready:
		timer.Stop()
		return true
	case <-timer.C:
		return false
	}
}
