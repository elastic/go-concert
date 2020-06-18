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

package timed

import (
	"context"
	"testing"
	"time"
)

func TestWait(t *testing.T) {
	t.Run("wait returns after the given period", func(t *testing.T) {
		start := time.Now()
		var duration = 250 * time.Millisecond
		err := Wait(context.TODO(), duration)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if waited := time.Since(start); waited < duration {
			t.Errorf("Expected to wait at least for %v, but did wait for %v", duration, waited)
		}
	})

	t.Run("wait returns with error on already cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		err := Wait(ctx, 10*time.Minute)
		if err == nil {
			t.Fatalf("Expected error")
		}
	})

	t.Run("wait returns early if context is cancelled in the meantime", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 250*time.Millisecond)
		defer cancel()
		err := Wait(ctx, 10*time.Minute)
		if err == nil {
			t.Fatalf("Expected Wait to return an error")
		}
	})
}

func TestPeriodic(t *testing.T) {
	t.Run("run until cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		count := 0
		const limit = 3
		Periodic(ctx, 10*time.Millisecond, func() {
			count++
			if count == limit {
				cancel()
			}
		})

		if count != limit {
			t.Fatalf("Function call counter missmatch. Want: %v, got: %v", limit, count)
		}
	})

	t.Run("do not run if context is already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		count := 0
		Periodic(ctx, 100*time.Millisecond, func() {
			count++
		})

		if count != 0 {
			t.Fatalf("Expected the periodic function to not be executed, but function was run %v times", count)
		}
	})
}
