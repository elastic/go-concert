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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMergeCancellation(t *testing.T) {
	mergers := map[string]func(a, b context.Context) (context.Context, context.CancelFunc){
		"MergeCancellation": func(a, b context.Context) (context.Context, context.CancelFunc) { return MergeCancellation(a, b) },
		"MergeContexts":     MergeContexts,
	}

	for name, merger := range mergers {
		t.Run(name, func(t *testing.T) {
			t.Run("cancelling context 1 cancels", func(t *testing.T) {
				ctx1, cancel1 := context.WithCancel(context.Background())
				ctx2, cancel2 := context.WithCancel(context.Background())
				defer cancel1()
				defer cancel2()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel := merger(ctx1, ctx2)
					defer cancel()
					<-ctx.Done()
				}()

				cancel1()
				wg.Wait() // <- deadlock if cancel signal was not distributed
			})

			t.Run("cancelling context 1 cancels", func(t *testing.T) {
				ctx1, cancel1 := context.WithCancel(context.Background())
				ctx2, cancel2 := context.WithCancel(context.Background())
				defer cancel1()
				defer cancel2()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel := merger(ctx1, ctx2)
					defer cancel()
					<-ctx.Done()
				}()

				cancel2()
				wg.Wait() // <- deadlock if cancel signal was not distributed
			})

			t.Run("canceller cancels new context", func(t *testing.T) {
				ctx1, cancel1 := context.WithCancel(context.Background())
				ctx2, cancel2 := context.WithCancel(context.Background())
				defer cancel1()
				defer cancel2()

				ctx, cancel := merger(ctx1, ctx2)
				cancel()
				<-ctx.Done()
			})

			t.Run("cancel if context 1 was canceled", func(t *testing.T) {
				defer goleak.VerifyNone(t)
				ctx1, cancelFn := context.WithCancel(context.Background())
				ctx2 := context.Background()
				cancelFn()

				ctx, cancel := merger(ctx1, ctx2)
				defer cancel()
				<-ctx.Done()
				assert.Error(t, ctx.Err())
			})

			t.Run("cancel if context 2 was canceled", func(t *testing.T) {
				defer goleak.VerifyNone(t)
				ctx1 := context.Background()
				ctx2, cancelFn := context.WithCancel(context.Background())
				cancelFn()

				ctx, cancel := merger(ctx1, ctx2)
				defer cancel()
				<-ctx.Done()
				assert.Error(t, ctx.Err())
			})

			t.Run("values are accessible", func(t *testing.T) {
				defer goleak.VerifyNone(t)
				ctx1 := contextWithValues("a", 1)
				ctx2, cancelFn := context.WithCancel(context.Background())
				cancelFn()

				ctx, cancel := merger(ctx1, ctx2)
				defer cancel()
				assert.Equal(t, 1, ctx.Value("a"))
			})
		})
	}
}

func TestMergeValues(t *testing.T) {
	type table map[interface{}]interface{}

	cases := map[string]struct {
		ctx        context.Context
		overwrites context.Context
		want       table
	}{
		"no values in overwrites": {
			ctx:        contextWithValues("a", 1),
			overwrites: contextWithValues(),
			want: table{
				"a": 1,
			},
		},
		"overwrite value on merge": {
			ctx:        contextWithValues("a", 1),
			overwrites: contextWithValues("a", 2),
			want: table{
				"a": 2,
			},
		},
		"values still accessible": {
			ctx:        contextWithValues("a", 1, "hello", "world"),
			overwrites: contextWithValues("a", 2, "answer", 42),
			want: table{
				"a":      2,
				"hello":  "world",
				"answer": 42,
			},
		},
	}

	mergers := map[string]func(a, b context.Context) context.Context{
		"MergeValues": func(a, b context.Context) context.Context { return MergeValues(a, b) },
		"MergeContexts": func(a, b context.Context) context.Context {
			ctx, cancel := MergeContexts(a, b)
			cancel()
			return ctx
		},
	}

	for name, merger := range mergers {
		t.Run(name, func(t *testing.T) {
			for name, test := range cases {
				t.Run(name, func(t *testing.T) {
					defer goleak.VerifyNone(t)

					ctx := merger(test.ctx, test.overwrites)
					actual := table{}
					for k := range test.want {
						actual[k] = ctx.Value(k)
					}

					assert.Equal(t, test.want, actual)
				})
			}
		})
	}
}

func TestMergeDeadline(t *testing.T) {
	ts1 := time.Now()
	ts2 := ts1.Add(1 * time.Hour)

	withDeadline := func(ts time.Time) context.Context {
		ctx, cancel := context.WithDeadline(context.Background(), ts)
		cancel()
		return ctx
	}

	cases := map[string]struct {
		ctx1 context.Context
		ctx2 context.Context
		want time.Time
	}{
		"no deadline": {
			ctx1: context.Background(),
			ctx2: context.Background(),
		},
		"first deadline": {
			ctx1: withDeadline(ts1),
			ctx2: withDeadline(ts2),
			want: ts1,
		},
		"second deadline": {
			ctx1: withDeadline(ts2),
			ctx2: withDeadline(ts1),
			want: ts1,
		},
		"first context only has deadline": {
			ctx1: withDeadline(ts1),
			ctx2: context.Background(),
			want: ts1,
		},
		"second context only has deadline": {
			ctx1: context.Background(),
			ctx2: withDeadline(ts1),
			want: ts1,
		},
	}

	mergers := map[string]func(a, b context.Context) context.Context{
		"MergeDeadline": func(a, b context.Context) context.Context { return MergeDeadline(a, b) },
		"MergeContexts": func(a, b context.Context) context.Context {
			ctx, cancel := MergeContexts(a, b)
			cancel()
			return ctx
		},
	}

	for name, merger := range mergers {
		t.Run(name, func(t *testing.T) {
			for name, test := range cases {
				t.Run(name, func(t *testing.T) {
					defer goleak.VerifyNone(t)

					ctx := merger(test.ctx1, test.ctx2)
					deadline, ok := ctx.Deadline()

					if test.want.IsZero() {
						assert.False(t, ok)
					} else {
						assert.True(t, ok)
						assert.Equal(t, test.want, deadline)
					}
				})
			}
		})
	}

}

func contextWithValues(args ...interface{}) context.Context {
	if len(args)%2 != 0 {
		panic("key values pairs incomplete")
	}

	ctx := context.Background()
	for i := 0; i < len(args); i += 2 {
		ctx = context.WithValue(ctx, args[i], args[i+1])
	}
	return ctx
}
