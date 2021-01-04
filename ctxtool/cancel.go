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

type cancelContext struct {
	canceller
}

// AutoCancel collects cancel functions to be executed at the end of the
// function scope.
//
// Example:
//   var ac AutoCancel
//   defer ac.Cancel()
//   ctx := ac.With(context.WithCancel(context.Background()))
//   ctx := ac.With(context.WithTimeout(ctx, 5 * time.Second))
//   ... // do something with ctx
type AutoCancel struct {
	funcs []context.CancelFunc
}

// Cancel calls all registered cancel functions in reverse order.
func (ac *AutoCancel) Cancel() {
	for _, fn := range ac.funcs {
		defer fn()
	}
}

// Add adds a new cancel function to the AutoCancel. The function will be run
// before any other already registered cancel function.
func (ac *AutoCancel) Add(fn context.CancelFunc) {
	ac.funcs = append(ac.funcs, fn)
}

// With is used to wrap a Context constructer call that returns a context and a
// cancel function.  The cancel function is automatically added to AutoCancel
// and the original context is returned as is.
func (ac *AutoCancel) With(ctx canceller, cancel context.CancelFunc) context.Context {
	ac.Add(cancel)
	return FromCanceller(ctx)
}

// FromCanceller creates a new context from a canceller. If a value that
// implements contex.Context is passed, then the value will be returned as is.
func FromCanceller(c canceller) context.Context {
	if ctx, ok := c.(context.Context); ok {
		return ctx
	}
	return cancelContext{c}
}

func (c cancelContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c cancelContext) Value(key interface{}) interface{} {
	return nil
}
