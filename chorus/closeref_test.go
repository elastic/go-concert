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

package chorus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloserRef(t *testing.T) {
	testPropageToChilds(t)
	testDontPropagateParent(t)
	testReturnErrWhenAlreadyClosed(t)
}

func testPropageToChilds(t *testing.T) {
	var parentCheck bool
	var childCheck bool
	parent := NewCloser(func() {
		parentCheck = true
	})

	_ = WithCloser(parent, func() {
		childCheck = true
	})

	parent.Close()
	assert.True(t, parentCheck)
	assert.True(t, childCheck)
}

func testDontPropagateParent(t *testing.T) {
	var parentCheck bool
	var childCheck bool
	var c int
	parent := NewCloser(func() {
		parentCheck = true
	})

	child := WithCloser(parent, func() {
		childCheck = true
		c++
	})

	child.Close()
	assert.False(t, parentCheck)
	assert.True(t, childCheck)
	assert.Equal(t, 1, c)

	// Child remove itself from the chain
	parent.Close()
	assert.Equal(t, 1, c)
}

func testReturnErrWhenAlreadyClosed(t *testing.T) {
	parent := NewCloser(nil)
	assert.Nil(t, parent.Err())
	parent.Close()
	assert.Equal(t, ErrClosed, parent.Err())
}
