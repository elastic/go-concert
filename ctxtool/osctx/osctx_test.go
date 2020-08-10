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
