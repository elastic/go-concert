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
)

func TestAutoCancel(t *testing.T) {
	t.Run("calls functions in reverse order", func(t *testing.T) {
		var values []int
		add := func(i int) func() {
			return func() { values = append(values, i) }
		}

		var ac AutoCancel
		ac.Add(add(1))
		ac.Add(add(2))
		ac.Add(add(3))
		ac.Cancel()
		assert.Equal(t, []int{3, 2, 1}, values)
	})

	t.Run("wraps and cancels context", func(t *testing.T) {
		var ac AutoCancel
		ctx := ac.With(context.WithCancel(context.Background()))
		ac.Cancel()
		assert.Error(t, ctx.Err())
	})
}
