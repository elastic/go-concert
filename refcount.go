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
	"sync/atomic"
)

// RefCount is an atomic reference counter. It can be used to track a shared
// resource it's lifetime and execute an action once it is clear the resource is
// not needed anymore.
//
// The zero value of RefCount is already in a valid state, which can be
// Released already.
type RefCount struct {
	Action func()
	count  uint32
	noCopy noCopy
}

// refCountFree indicates when a RefCount.Release shall return true.  It's
// chosen such that the zero value of RefCount is a valid value which will
// return true if Release is called without calling Retain before.
const refCountFree uint32 = ^uint32(0)
const refCountOops uint32 = refCountFree - 1

// Retain increases the ref count.
func (c *RefCount) Retain() {
	x := atomic.AddUint32(&c.count, 1)
	if x == 0 {
		panic("retaining released ref count")
	}
}

// Release decreases the reference count. It returns true, if the reference count
// has reached a 'free' state.
// Releasing a reference count in a free state will trigger a panic.
// If an Action is configured, then this action will be run once the
// refcount becomes free.
func (c *RefCount) Release() bool {
	x := atomic.AddUint32(&c.count, ^uint32(0))
	switch {
	case x == refCountFree:
		if c.Action != nil {
			c.Action()
		}
		return true
	case x == refCountOops:
		panic("ref count released too often")
	default:
		return false
	}
}
