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

import "time"

type CHMutex struct {
	ch chan struct{}
}

type doneContext interface {
	Done() <-chan struct{}
	Err() error
}

func MakeCHMutex() CHMutex {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return CHMutex{ch: ch}
}

func (c CHMutex) Lock() {
	<-c.ch
}

func (c CHMutex) LockTimeout(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	select {
	case <-c.ch:
		timer.Stop()
		return true
	case <-timer.C:
		select {
		case <-c.ch: // still lock, if timer and lock occured at the same time
			return true
		default:
			return false
		}
	}
}

func (c CHMutex) LockContext(context doneContext) error {
	select {
	case <-c.ch:
		return nil
	case <-context.Done():
		return context.Err()
	}
}

func (c CHMutex) Await() <-chan struct{} {
	return c.ch
}

func (c CHMutex) Unlock() {
	c.ch <- struct{}{}
}
