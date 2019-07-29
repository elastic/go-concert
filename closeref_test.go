package concert

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
