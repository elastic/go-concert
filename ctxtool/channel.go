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
	"time"
)

type chanCanceller <-chan struct{}

type chanContext <-chan struct{}

// WithChannel creates a context that is cancelled if the parent context is cancelled
// or if the given channel is closed.
func WithChannel(parent context.Context, ch <-chan struct{}) (context.Context, context.CancelFunc) {
	return MergeCancellation(parent, chanCanceller(ch))
}

// FromChannel creates a new context from a channel.
func FromChannel(ch <-chan struct{}) context.Context {
	return chanContext(ch)
}

func (c chanCanceller) Done() <-chan struct{} {
	return (<-chan struct{})(c)
}

func (c chanCanceller) Err() error {
	select {
	case <-c:
		return context.Canceled
	default:
		return nil
	}
}

func (c chanContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c chanContext) Done() <-chan struct{} {
	return c
}

// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (c chanContext) Err() error {
	select {
	case <-c:
		return context.Canceled
	default:
		return nil
	}
}

func (c chanContext) Value(key interface{}) interface{} {
	return nil
}
