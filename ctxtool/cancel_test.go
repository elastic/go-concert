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
