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

package concert_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-concert"
)

func TestRefCount(t *testing.T) {
	t.Run("create and release", func(t *testing.T) {
		var released bool
		var r concert.RefCount
		if r.Release() {
			released = true
		}

		assert.True(t, released)
	})

	t.Run("release with action", func(t *testing.T) {
		var released bool
		r := concert.RefCount{
			Action: func(err error) { released = true },
		}

		assert.True(t, r.Release())
		assert.True(t, released)
	})

	t.Run("releasing too often panics", func(t *testing.T) {
		assert.Panics(t, func() {
			var r concert.RefCount
			r.Release()
			r.Release()
		})
	})

	t.Run("retain on released refcount panics", func(t *testing.T) {
		assert.Panics(t, func() {
			var r concert.RefCount
			r.Release()
			r.Retain()
		})
	})

	t.Run("fail passes error releases the refcount", func(t *testing.T) {
		var released bool
		errTest := errors.New("test")
		r := concert.RefCount{
			Action: func(err error) {
				assert.Equal(t, errTest, err)
				released = true
			},
		}

		assert.True(t, r.Fail(errTest))
		assert.Equal(t, errTest, r.Err())
		assert.True(t, released)
	})

	t.Run("fail stores first error only by default", func(t *testing.T) {
		errTest := errors.New("test")
		var r concert.RefCount
		r.Retain()
		assert.False(t, r.Fail(errTest))
		assert.True(t, r.Fail(errors.New("other")))
		assert.Equal(t, errTest, r.Err())
	})

	t.Run("OnError callback properly manipluates error", func(t *testing.T) {
		r := concert.RefCount{
			OnError: func(old, new error) error {
				if old == nil {
					return new
				}
				return fmt.Errorf("%s: %s", new, old)
			},
		}
		r.Retain()
		assert.False(t, r.Fail(errors.New("error1")))
		assert.True(t, r.Fail(errors.New("error2")))
		assert.Equal(t, "error2: error1", r.Err().Error())
	})
}
