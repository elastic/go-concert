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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnceSignaler(t *testing.T) {
	t.Run("triggering provides a signal on the done channel", func(t *testing.T) {
		s := NewOnceSignaler()
		s.Trigger()
		<-s.Done() // <- deadlock if trigger was not effective
	})

	t.Run("triggering multiple times does not panic", func(t *testing.T) {
		s := NewOnceSignaler()
		s.Trigger()
		s.Trigger()
	})

	t.Run("no error if not triggered", func(t *testing.T) {
		s := NewOnceSignaler()
		assert.NoError(t, s.Err())
	})

	t.Run("report Canceled error when triggered", func(t *testing.T) {
		s := NewOnceSignaler()
		s.Trigger()
		assert.Equal(t, Canceled, s.Err())
	})

	t.Run("callback is executed when triggered", func(t *testing.T) {
		s := NewOnceSignaler()
		count := 0
		s.OnSignal(func() { count++ })
		s.Trigger()
		assert.Equal(t, 1, count)
	})

	t.Run("callback is called at most once", func(t *testing.T) {
		s := NewOnceSignaler()
		count := 0
		s.OnSignal(func() { count++ })
		s.Trigger()
		s.Trigger()
		s.Trigger()
		assert.Equal(t, 1, count)
	})
}
