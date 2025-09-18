package main

import (
	"sync"
	"time"
)

// DoAfterUnlessCanceled waits for the duration to elapse and then calls f unless cancel was called first.
// It is like time.AfterFunc, except that it returns a function that either prevents f from being called
// or waits for f to complete (if it has already been called) before returning.
// cancel() is idempotent and returns true if cancellation succeeded (i.e., f did not start),
// or false if f was called (i.e. d elapsed before cancel was called).
func DoAfterUnlessCanceled(d time.Duration, f func()) (cancel func() bool) {
	canceled := make(chan struct{})
	var (
		once sync.Once
		wg   sync.WaitGroup
	)
	t := time.NewTimer(d)
	wg.Add(1)
	var fCanceled bool
	go func() {
		defer wg.Done()
		select {
		case <-t.C:
			f()
		case <-canceled:
			fCanceled = true
		}
	}()

	return func() bool {
		once.Do(func() { close(canceled) })
		wg.Wait()
		return fCanceled
	}
}
