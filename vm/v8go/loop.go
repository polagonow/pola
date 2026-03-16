// Package v8go provides a V8-backed VM implementation for the GoJSX framework.
//
// Each VM owns one V8 isolate and a goroutine-pinned event loop. The loop
// drains V8 microtasks (Promise continuations) after every task so that
// async server components resolve correctly.
package v8go

import "runtime"

// eventLoop pins all V8 operations onto a single OS thread.
// After every task it calls ctx.PerformMicrotaskCheckpoint() so that
// Promise continuations (async/await continuations) run between calls.
type eventLoop struct {
	taskCh     chan func()
	stopCh     chan struct{}
	checkpoint func() // set to ctx.PerformMicrotaskCheckpoint once context is created
}

func newEventLoop() *eventLoop {
	el := &eventLoop{
		taskCh: make(chan func(), 256),
		stopCh: make(chan struct{}),
	}
	go el.run()
	return el
}

// setCheckpoint provides the microtask drain function once the V8 context exists.
// Must be called from inside a RunSync block (i.e. on the event loop goroutine).
func (el *eventLoop) setCheckpoint(fn func()) {
	el.checkpoint = fn
}

func (el *eventLoop) run() {
	runtime.LockOSThread()
	for {
		select {
		case task := <-el.taskCh:
			task()
			if el.checkpoint != nil {
				el.checkpoint()
			}
		case <-el.stopCh:
			return
		}
	}
}

// RunSync submits fn to the event loop goroutine and blocks until it completes.
func (el *eventLoop) RunSync(fn func()) {
	done := make(chan struct{})
	el.taskCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// RunAsync queues fn on the event loop without blocking the caller.
// Used by bridge-function goroutines to resolve/reject Promises.
func (el *eventLoop) RunAsync(fn func()) {
	el.taskCh <- fn
}

// Stop terminates the event loop goroutine.
func (el *eventLoop) Stop() {
	close(el.stopCh)
}

