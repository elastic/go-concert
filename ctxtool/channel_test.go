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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestWithChannel(t *testing.T) {
	t.Run("cancel if channel is closed", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		ch := make(chan struct{})
		ctx := WithChannel(context.Background(), ch)
		assert.NoError(t, ctx.Err())
		close(ch)
		<-ctx.Done()
		assert.Error(t, ctx.Err())
	})

	t.Run("cancel if parent context is cancelled", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		ctx, cancelFn := context.WithCancel(context.Background())
		ch := make(chan struct{})
		defer cancelFn()
		ctx = WithChannel(ctx, ch)
		cancelFn()
		<-ctx.Done()
		assert.Error(t, ctx.Err())
	})

	t.Run("values are accessible", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ch := make(chan struct{})
		ctx := context.WithValue(context.Background(), "hello", "world")
		ctx = WithChannel(ctx, ch)

		defer func() {
			close(ch)
			<-ctx.Done()
		}()

		assert.Equal(t, "world", ctx.Value("hello"))
	})
}
