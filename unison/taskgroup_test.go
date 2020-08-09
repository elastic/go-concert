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

package unison

import (
	"context"
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

	t.Run("cancel is no error", func(t *testing.T) {
		var grp TaskGroup
		grp.Go(func(_ Canceler) error { return context.Canceled })
		require.NoError(t, grp.Stop())
	})

	t.Run("cancel does not trigger stop", func(t *testing.T) {
		count := 0
		grp := TaskGroup{
			StopOnError: func(_ error) bool { count++; return false },
		}
		grp.Go(func(_ Canceler) error { return context.Canceled })
		grp.Stop()

		require.Equal(t, 0, count)
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
