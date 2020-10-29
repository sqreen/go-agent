// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsafe

// Go mimics the `go` goroutine built-in to execute function `f` in a goroutine
// but with the ability to safely recover from any panic occurring while it
// executes. To do so, it uses `Call()` and returns an error channel in order
// to retrieve any panic occurring during the execution of `f()` or the error
// it returns otherwise. An error is sent into the channel only in case of
// error or panic, and is closed in any case before returning from the
// goroutine.
//
// Usage example:
//
//		errChan := safe.Go(f)
//    // ...
//		select {
//			case err := <-errChan:
//				var panicErr *sqlib.PanicError
//				if xerrors.As(err, &panicErr) {
//					// A panic occurred while executing f()
//				} else {
//					// A regular error was returned by f()
//				}
//			// ...
//		}
//
func Go(f func() error, c chan error) {
	go func() {
		err := Call(f)
		if c != nil {
			c <- err
		}
	}()
}
