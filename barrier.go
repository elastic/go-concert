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

import "sync"

// Barrier implements a synchronisation barrier.
// The barrier blocks until all participants have reached `(*Barrier).Wait`.
// On Wait, all participants are unblocked and the barrier is reset.
// Participants can be dynamically added/removed to the barrier using Attach
// and Detach.
//
// A Barrier must be creating using NewBarrier.
type Barrier struct {
	mu   sync.Mutex
	cond *sync.Cond

	participants uint
	arrived      uint
	phase        uint
	elected      uint
}

// NewBarrier creates a new barrier with a set number of participants.
// If the number of participants is 0, no call to Wait or Detach is allowed.
func NewBarrier(participants uint) *Barrier {
	b := &Barrier{}
	b.cond = sync.NewCond(&b.mu)
	b.participants = participants
	return b
}

// Attach adds a new participant to the barrier.
// Detach must be used to unsubscribe from the barrier.
func (b *Barrier) Attach() {
	b.mu.Lock()
	b.participants++
	b.mu.Unlock()
}

// Detach removes a participant from the barrier. It returns true if the last
// participant is returned.
func (b *Barrier) Detach() (last bool) {
	b.mu.Lock()

	if b.participants == 0 {
		b.mu.Unlock()
		panic("detach from barrier without participants")
	}

	b.participants--

	var release bool
	if b.participants > 0 && b.participants == b.arrived {
		release = true
		b.arrived = 0
		b.phase++
	}
	last = b.participants == 0
	b.mu.Unlock()

	if release {
		b.cond.Broadcast()
	}

	return last
}

// Wait blocks until all participants have reached the next call to Wait.  One
// go-routine is elected to reset the barrier. Wait returns true for the
// elected go-routine, which can be used to signal some work only one
// go-routine shall execute after Wait. For example:
//
//    for {
//        ...
//        if b.Wait() {
//            buffer.Flush()
//            file.Sync()
//        }
//    }
//
func (b *Barrier) Wait() bool {
	var release bool

	b.mu.Lock()

	if b.participants == 0 {
		b.mu.Unlock()
		panic("wait on barrier without participants")
	}

	phaseStart := b.phase
	phaseNext := phaseStart + 1
	b.arrived++
	if b.arrived == b.participants {
		release = true
		b.arrived = 0
		b.phase = phaseNext
		b.elected = phaseNext
	}

	if release {
		// last participant arrived -> signal all participants to continue
		b.mu.Unlock()
		b.cond.Broadcast()
		return true
	}

	var elected bool
	for {
		release = b.phase == phaseNext
		if release && b.elected != phaseNext {
			// re-elect one worker in case the last worker unblocking wait was detached.
			b.elected = b.phase
			elected = true
		}

		if release {
			break
		}

		b.cond.Wait()
	}

	b.mu.Unlock()
	return elected
}

// Participants reports the number of participants
func (b *Barrier) Participants() uint {
	b.mu.Lock()
	participants := b.participants
	b.mu.Unlock()
	return participants
}
