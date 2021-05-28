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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskGroup(t *testing.T) {
	t.Run("stop sends signal to worker", func(t *testing.T) {
		var grp TaskGroup
		wg, wgStart := wgCount(1), wgCount(1)
		err := grp.Go(func(cancel context.Context) error {
			defer wg.Done()
			wgStart.Done()
			<-cancel.Done()
			return nil
		})
		require.NoError(t, err)

		wgStart.Wait()
		err = grp.Stop()
		require.NoError(t, err)
		wg.Wait()
	})

	t.Run("cancel is no error", func(t *testing.T) {
		var grp TaskGroup
		grp.Go(func(_ context.Context) error { return context.Canceled })
		require.NoError(t, grp.Stop())
	})

	t.Run("can not create go-routine if group has been stopped", func(t *testing.T) {
		var grp TaskGroup
		grp.Stop()
		require.Equal(t, ErrGroupClosed, grp.Go(func(_ context.Context) error { return nil }))
	})

	t.Run("signal shutdown via context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		grp := TaskGroupWithCancel(ctx)

		wg, wgStart := wgCount(1), wgCount(1)
		grp.Go(func(c context.Context) error {
			defer wg.Done()
			wgStart.Done()
			<-c.Done()
			return nil
		})

		wgStart.Wait()
		cancel()
		wg.Wait()
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("stop is propogate", func(t *testing.T) {
			var tg TaskGroup
			ctx := tg.Context()
			tg.Stop()
			require.Equal(t, context.Canceled, ctx.Err())
		})

		t.Run("parent context shutdown is propagated", func(t *testing.T) {
			parentCtx, cancel := context.WithCancel(context.TODO())
			tg := TaskGroupWithCancel(parentCtx)
			ctx := tg.Context()
			cancel()
			require.Equal(t, context.Canceled, ctx.Err())
		})
	})
}

func TestTaskGroup_MaxErrors(t *testing.T) {

	const numErrors = 5
	const limit = 3
	tg := TaskGroup{MaxErrors: limit, OnQuit: ContinueOnErrors}

	var errs [numErrors]error
	for i := 0; i < numErrors; i++ {
		errs[i] = fmt.Errorf("opps: %v", i)
	}

	for _, err := range errs {
		wg := wgCount(1)
		tg.Go(func(_ context.Context) error {
			defer wg.Done()
			return err
		})
		wg.Wait()
	}

	want := errs[numErrors-limit:]
	got := tg.waitErrors()
	require.Equal(t, want, got)
}

func TestTaskgroup_OnQuit_ContinueOnError(t *testing.T) {
	onQuit := ContinueOnErrors

	t.Run("continue on normal return", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, finishedGroupWorker)
	})

	t.Run("continues on error", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, failingGroupWorker)
	})
}

func TestTaskgroup_OnQuit_RestartOnError(t *testing.T) {
	onQuit := RestartOnError

	t.Run("continue on normal return", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, finishedGroupWorker)
	})

	t.Run("restarts on error", func(t *testing.T) {
		var count int
		grp := TaskGroup{OnQuit: onQuit}

		grp.Go(func(_ context.Context) error {
			count++
			t.Log(count)
			if count == 1 {
				return errors.New("oops")
			}
			return nil
		})

		grp.Wait()
		require.Equal(t, 2, count)
	})

}

func TestTaskgroup_OnQuit_StopAll(t *testing.T) {
	onQuit := StopAll

	t.Run("stops on normal return", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, finishedGroupWorker)
	})

	t.Run("stop on cancel", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, internalCancelGroupWorker)
	})

	t.Run("stop on error", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, failingGroupWorker)
	})
}

func TestTaskgroup_OnQuit_StopOnError(t *testing.T) {
	onQuit := StopOnError

	t.Run("continue on normal return", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, finishedGroupWorker)
	})

	t.Run("continue on cancel", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, internalCancelGroupWorker)
	})

	t.Run("stop on error", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, failingGroupWorker)
	})
}

func TestTaskgroup_OnQuit_StopOnErrorOrCancel(t *testing.T) {
	onQuit := StopOnErrorOrCancel

	t.Run("continue on normal return", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupContinuesIf(t, &grp, finishedGroupWorker)
	})

	t.Run("stop on cancel", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, internalCancelGroupWorker)
	})

	t.Run("stop on error", func(t *testing.T) {
		grp := TaskGroup{OnQuit: onQuit}
		testTaskGroupStopsIf(t, &grp, failingGroupWorker)
	})
}

func testTaskGroupContinuesIf(t *testing.T, grp *TaskGroup, fnFirst func(context.Context) error) {
	wgFirst, wgSecondStart, wgSecondDone := wgCount(1), wgCount(1), wgCount(1)
	grp.Go(func(ctx context.Context) error {
		defer wgFirst.Done()
		return fnFirst(ctx)
	})

	wgFirst.Wait()
	require.NoError(t, grp.Go(func(c context.Context) error {
		defer wgSecondDone.Done()
		wgSecondStart.Done()
		<-c.Done()
		return nil
	}))

	wgSecondStart.Wait()
	grp.Stop()
	wgSecondDone.Wait()
}

func testTaskGroupStopsIf(t *testing.T, grp *TaskGroup, fnFail func(context.Context) error) {
	wgStart := wgCount(1)
	grp.Go(func(ctx context.Context) error {
		wgStart.Done()
		<-ctx.Done()
		return nil
	})

	grp.Go(func(ctx context.Context) error {
		wgStart.Wait()
		return fnFail(ctx)
	})

	grp.Wait()
}

func wgCount(n int) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	return &wg
}

func failingGroupWorker(_ context.Context) error {
	return errors.New("oops")
}

func finishedGroupWorker(_ context.Context) error {
	return nil
}

func internalCancelGroupWorker(_ context.Context) error {
	return context.Canceled
}
