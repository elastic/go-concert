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

package ctxtool

import (
	"context"
	"sync"
	"time"
)

type channelCtx struct {
	parent context.Context
	ch     <-chan struct{}

	mu  sync.Mutex
	err error
}

var closedChan = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// WithChannel creates a context that is cancelled if the parent context is cancelled
// or if the given channel is closed.
func WithChannel(parent context.Context, ch <-chan struct{}) context.Context {
	if err := parent.Err(); err != nil {
		return &channelCtx{
			parent: parent,
			ch:     closedChan,
			err:    err,
		}
	}

	select {
	case <-ch:
		return &channelCtx{parent: parent, ch: closedChan, err: context.Canceled}
	default:
	}

	chDone := make(chan struct{})
	ctx := &channelCtx{
		parent: parent,
		ch:     chDone,
	}
	go ctx.waitCancel(chDone, ch)
	return ctx
}

func (c *channelCtx) waitCancel(chDone chan struct{}, chIn <-chan struct{}) {
	var err error
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.err = err
		close(chDone)
	}()

	select {
	case <-c.parent.Done():
		err = c.parent.Err()
	case <-chIn:
		err = context.Canceled
	}
}

func (c *channelCtx) Deadline() (deadline time.Time, ok bool) {
	if c.parent != nil {
		return c.parent.Deadline()
	}
	return deadline, ok
}

func (c *channelCtx) Done() <-chan struct{} {
	return c.ch
}

func (c *channelCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *channelCtx) Value(key interface{}) interface{} {
	return c.parent.Value(key)
}
