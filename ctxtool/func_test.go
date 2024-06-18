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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestWithFunc(t *testing.T) {
	t.Run("executed on cleanup 'cancel' call", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		var count atomic.Int64
		wg := makeWaitGroup((1)) // func is always run asynchronously, wait
		ctx, cancel := WithFunc(context.Background(), func() {
			defer wg.Done()
			count.Add(1)
		})
		cancel()
		wg.Wait()
		assert.NotNil(t, ctx)
		assert.Equal(t, int64(1), count.Load())
	})

	t.Run("executed func on cancel", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		var (
			ctx              context.Context
			cancel1, cancel2 context.CancelFunc
		)

		done := make(chan struct{})
		var count atomic.Int64
		ctx, cancel1 = context.WithCancel(context.Background())
		ctx, cancel2 = WithFunc(ctx, func() {
			count.Add(1)
			close(done)
		})
		defer cancel2()
		cancel1()
		<-done
		assert.Equal(t, int64(1), count.Load())
	})

	t.Run("wait for other before we continue cancelling", func(t *testing.T) {
		ctx1, canceler := context.WithCancel(context.Background())
		defer canceler()

		var mu sync.Mutex
		var values []int

		wg := makeWaitGroup(2)
		wg1 := makeWaitGroup(1)
		go func() {
			defer wg1.Done()
			defer wg.Done()
			<-ctx1.Done()

			mu.Lock()
			defer mu.Unlock()
			values = append(values, 1)
		}()

		// create context that waits for wg1 to finish
		ctx2, cancel2 := WithFunc(ctx1, wg1.Wait)
		defer cancel2()
		go func() {
			defer wg.Done()
			<-ctx2.Done()

			mu.Lock()
			defer mu.Unlock()
			values = append(values, 2)
		}()

		// get the machinery run:
		//   - start go-routine 1 and 2
		//   - cancel top-level context
		//   -> go-routine 1 will shutdown
		//     - signal propagation in ctx2 is blocked until wg1.Wait unblocks
		//   -> go-routine calls wg1.Done -> signal propagation continues
		//   -> go-routine 2 will shutdown
		//     - signal all helper go-routines are done

		canceler()
		wg.Wait()
		assert.Equal(t, []int{1, 2}, values)
	})
}

func makeWaitGroup(i int) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(i)
	return &wg
}
