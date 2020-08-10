package osctx

import (
	"context"
	"os"
	"os/signal"
)

// WithSignal creates a context that will be cancelled if any of the configured
// signals is received by the process. The signal handler will be removed automatically in case the parent context
// gets cancelled or when the cancel function is called.
//
// The context should be used to trigger application shutdown. If the signal is
// received again, the signal handler will force shutdown the process with exit
// code 3.
//
// example:
//
//  func main() {
//		ctx, cancel := osctx.WithSignal(context.Background(), os.Kill)
//		defer cancel()
//
//		for ctx.Err == nil {
//			// main run loop
//		}
//  }
func WithSignal(ctx context.Context, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 1)
	go func() {
		defer func() {
			signal.Stop(ch)
			cancel()
		}()

		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
			// force shutdown in case we receive another signal
			<-ch
			os.Exit(3)
		}
	}()

	signal.Notify(ch, sigs...)
	return ctx, cancel
}
