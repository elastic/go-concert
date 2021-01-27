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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCell(t *testing.T) {
	t.Run("read state from init", func(t *testing.T) {
		cell := NewCell("init")
		assert.Equal(t, "init", cell.Get())
	})

	t.Run("sync update cell", func(t *testing.T) {
		cell := NewCell("init")
		cell.Set("test")
		assert.Equal(t, "test", cell.Get())
	})

	t.Run("Wait does not block after set", func(t *testing.T) {
		cell := NewCell("init")
		cell.Set("test")

		val, err := cell.Wait(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, "test", val)
	})

	t.Run("cancel wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		cell := NewCell("init")
		_, err := cell.Wait(ctx)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("wait for update", func(t *testing.T) {
		cell := NewCell("init")

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
			cell.Set("test")
		}()

		val, err := cell.Wait(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "test", val)
	})
}

// ExampleCellACK tracks the number of ACKed events without backpressure in the
// generating thread, even if the consumer is blocked. The consumer computes
func ExampleCell_acking() {
	type exampleACKer struct {
		state      *Cell
		ackedWrite uint
		ackedRead  uint
	}

	// exampleACK acks a single event by updating the 'absolute' state.
	// The function return immediately, even if the "reader" process is
	// blocking for a long time.
	exampleACK := func(acker *exampleACKer) {
		acker.ackedWrite++
		acker.state.Set(acker.ackedWrite)
	}

	// exampleWaitACKed waits for state changes and returns the number of
	// events that have been acked since the last read. It returns the accumulated state.
	exampleWaitACKed := func(acker *exampleACKer, ctx context.Context) (uint, error) {
		st, err := acker.state.Wait(ctx)
		if err != nil {
			return 0, err
		}

		v := st.(uint)
		acker.ackedRead, v = v, v-acker.ackedRead
		return v, nil
	}

	const max = 100
	acker := &exampleACKer{state: NewCell(0)}

	// start go-routine that ACKs single events
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() { // ACKer thread
		defer wg.Done()

		// We ACK event by event, but merge the overall state
		// by reporting the absolute value.
		for send := 0; send < max; send++ {
			exampleACK(acker)
		}
	}()

	// reader loop
	var totalACKed uint
	for totalACKed < max {
		acked, _ := exampleWaitACKed(acker, context.TODO())

		// Handle 'N" events being ACKed
		totalACKed += acked
	}

	fmt.Println("Total:", totalACKed)
	// Output: Total: 100
}
