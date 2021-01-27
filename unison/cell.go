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

// Cell stores some state of type interface{}.
// Intermittent updates are lost, in case the Cell is updated faster than the
// consumer tries to read for state updates. Updates are immediate, there will
// be no backpressure applied to producers.
//
// In case the Cell is used to transmit updates without backpressure, the
// absolute state must be computed by the producer beforehand.
//
// A typical use-case for cell is to generate asynchronous configuration updates (no deltas).
type Cell struct {
	// All writes/reads to any of the internal fields must be guarded by mu.
	mu Mutex

	// logical config state update counters.
	// The readID always follows writeID. We are using the most recent state
	// update if readID == waitID.
	writeID uint64
	readID  uint64

	state interface{}

	waiter chan struct{}
}

// NewCell creates a new call instance with its initial state. Subsequent reads
// will return this state, if there have been no updates.
func NewCell(st interface{}) *Cell {
	return &Cell{
		mu:    MakeMutex(),
		state: st,
	}
}

// Get returns the current state.
func (c *Cell) Get() interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read()
}

// Wait blocks until it an update since the last call to Get or Wait has been found.
// The cancel context can be used to interrupt the call to Wait early. The
// error value will be set to the value returned by cancel.Err() in case Wait
// was interrupted. Wait does not produce any errors that need to be handled by itself.
func (c *Cell) Wait(cancel Canceler) (interface{}, error) {
	c.mu.Lock()

	if c.readID != c.writeID {
		defer c.mu.Unlock()
		return c.read(), nil
	}

	var waiter chan struct{}
	if c.waiter == nil {
		waiter = make(chan struct{})
		c.waiter = waiter
	} else {
		waiter = c.waiter
	}
	c.mu.Unlock()

	select {
	case <-cancel.Done():
		// we don't bother to check the waiter channel again. Cancellation if
		// detected has priority.
		c.mu.Lock()
		defer c.mu.Unlock()
		c.waiter = nil
		return nil, cancel.Err()
	case <-waiter:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.read(), nil
	}
}

// Set updates the state of the Cell and unblocks a waiting consumer.
// Set does not block.
func (c *Cell) Set(st interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writeID++
	c.state = st

	if c.waiter != nil {
		close(c.waiter)
		c.waiter = nil
	}
}

func (c *Cell) read() interface{} {
	c.readID = c.writeID
	return c.state
}
