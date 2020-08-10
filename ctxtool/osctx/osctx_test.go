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

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package osctx

import (
	"context"
	"os"
	"syscall"
	"testing"
)

func TestWithSignal(t *testing.T) {
	t.Run("quit if parent context is cancelled", func(t *testing.T) {
		parent, cancel := context.WithCancel(context.Background())
		cancel()

		ctx, cancel := WithSignal(parent, os.Interrupt)
		defer cancel()

		<-ctx.Done() // must not block
	})

	t.Run("return on explicit cancel", func(t *testing.T) {
		ctx, cancel := WithSignal(context.Background(), os.Interrupt)
		cancel()
		<-ctx.Done() // must not block
	})

	t.Run("quit on signal", func(t *testing.T) {
		testSignal := syscall.SIGUSR1

		ctx, cancel := WithSignal(context.Background(), testSignal)
		defer cancel()

		syscall.Kill(syscall.Getpid(), testSignal)
		<-ctx.Done() // must not block, as the signal has been delivered.
	})
}
