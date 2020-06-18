package timed

import "time"

type canceler interface {
	Done() <-chan struct{}
	Err() error
}

// Wait blocks for the configuration duration or until the passed context
// signal canceling.
// Wait return ctx.Err() if the context got cancelled early. If the duration
// has passed without the context being cancelled, Wait returns nil.
//
// Example:
//   fmt.Printf("wait for 5 seconds...")
//   if err := Wait(ctx, 5 * time.Second); err != nil {
//       fmt.Printf("shutting down")
//       return err
//   }
//   fmt.Println("done")
func Wait(ctx canceler, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Periodic executes fn on every period. Periodic returns if the context is
// cancelled.
// The underlying ticket adjusts the intervals or drops ticks to make up for
// slow runs of fn. If fn is active, Peridoc will only return when fn has
// finished.
// The period must be greater than 0, otherwise Periodic panics.
func Periodic(ctx canceler, period time.Duration, fn func()) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	done := ctx.Done()
	for {
		// always check for cancel first, to not accidentily trigger another run if
		// the context is already cancelled, but we have already received another
		// ticker signal
		select {
		case <-done:
			return
		default:
		}

		select {
		case <-ticker.C:
			fn()
		case <-done:
			return
		}
	}
}
