package unison

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiErrGroup(t *testing.T) {
	t.Run("returns empty list if no go-routine was started", func(t *testing.T) {
		var grp MultiErrGroup
		assert.Equal(t, 0, len(grp.Wait()))
	})

	t.Run("returns empty list if no go-routine failed", func(t *testing.T) {
		var grp MultiErrGroup
		grp.Go(func() error { return nil })
		assert.Equal(t, 0, len(grp.Wait()))
	})

	t.Run("Returns multiple errors", func(t *testing.T) {
		var grp MultiErrGroup
		grp.Go(func() error { return errors.New("1") })
		grp.Go(func() error { return errors.New("2") })
		assert.Equal(t, 2, len(grp.Wait()))
	})
}
