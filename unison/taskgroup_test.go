package unison

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskGroup(t *testing.T) {
	t.Run("stop sends signal to worker", func(t *testing.T) {
		var grp TaskGroup
		ch := make(chan bool, 1)
		err := grp.Go(func(cancel Canceler) error {
			<-cancel.Done()
			ch <- true
			return nil
		})
		require.NoError(t, err)

		err = grp.Stop()
		require.NoError(t, err)
		<-ch // this blocks if works did not shut down
	})

	t.Run("can not create go-routine if group has been stopped", func(t *testing.T) {
		var grp TaskGroup
		grp.Stop()
		require.Equal(t, ErrGroupClosed, grp.Go(func(_ Canceler) error { return nil }))
	})

	t.Run("stop all tasks on error", func(t *testing.T) {
		grp := TaskGroup{
			StopOnError: func(_ error) bool { return true },
		}

		ch := make(chan bool, 2)
		grp.Go(func(c Canceler) error {
			<-c.Done()
			ch <- true
			return nil
		})
		grp.Go(func(c Canceler) error {
			ch <- true
			return errors.New("oops")
		})

		<-ch
		<-ch // block if not all workers have been shut down

		// send stop to collect errors
		require.Error(t, grp.Stop())
	})
}
