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

type mergeCancelCtx struct {
	ctx1, ctx2 context.Context
	ch         <-chan struct{}

	mu  sync.Mutex
	err error
}

type mergeValueCtx struct {
	context.Context
	overwrites context.Context
}

// MergeContexts merges cancallation and values of 2 contexts.
// The resulting context is cancelled by the first context that got cancalled.
// The ctx2 overwrites values in ctx1 during value lookup.
func MergeContexts(ctx1, ctx2 context.Context) context.Context {
	return MergeValues(MergeCancellation(ctx1, ctx2), ctx2)
}

// MergeCancellation creates a new context that will be cancelled if one of the
// two input contexts gets cancalled. The `Values` method of the new context only
// uses values from `ctx`. With MergeValues, in order to merge values only.
func MergeCancellation(ctx, other context.Context) context.Context {
	err := ctx.Err()
	if err == nil {
		err = other.Err()
	}
	if err != nil {
		// at least one context is already cancelled
		return &mergeCancelCtx{
			ctx1: ctx,
			ctx2: other,
			ch:   closedChan,
			err:  err,
		}
	}

	if ctx.Done() == nil && other.Done() == nil {
		// context is never cancelled.
		return &mergeCancelCtx{
			ctx1: ctx,
			ctx2: other,
		}
	}

	chDone := make(chan struct{})
	merged := &mergeCancelCtx{
		ctx1: ctx,
		ctx2: other,
		ch:   chDone,
	}
	go merged.waitCancel(chDone)
	return merged
}

// MergeValues merges the values from ctx and overwrites. Value lookup will occur on `overwrites` first.
// Deadline and cancellation are still driven by the first context. In order to merge cancellation use
// MergeCancellation.
func MergeValues(ctx, overwrites context.Context) context.Context {
	return &mergeValueCtx{ctx, overwrites}
}

func (c *mergeCancelCtx) waitCancel(chDone chan struct{}) {
	var err error
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.err = err
		close(chDone)
	}()

	select {
	case <-c.ctx1.Done():
		err = c.ctx1.Err()
	case <-c.ctx2.Done():
		err = c.ctx2.Err()
	}
}

func (c *mergeCancelCtx) Deadline() (deadline time.Time, ok bool) {
	d1, ok1 := c.ctx1.Deadline()
	d2, ok2 := c.ctx2.Deadline()
	if !ok1 {
		return d2, ok2
	} else if !ok2 {
		return d1, ok1
	}

	if d1.Before(d2) {
		return d1, true
	}
	return d2, true
}

func (c *mergeCancelCtx) Done() <-chan struct{} {
	return c.ch
}

func (c *mergeCancelCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *mergeCancelCtx) Value(key interface{}) interface{} {
	return c.ctx1.Value(key)
}

func (c *mergeValueCtx) Value(key interface{}) interface{} {
	if val := c.overwrites.Value(key); val != nil {
		return val
	}
	return c.Context.Value(key)
}
