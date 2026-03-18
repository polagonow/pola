// Package eventloop provides a goroutine-serialised event loop shared by all
// engine backends (sobek, v8go, qjs). All JS operations must run on the same
// goroutine to satisfy each engine's single-threaded invariant.
//
// Usage:
//
//	loop := eventloop.New(false)               // or true for V8 (LockOSThread)
//	loop.RunSync(func() {
//	    // create VM, then:
//	    loop.SetCheckpoint(func() { /* drain microtasks */ })
//	})
//	loop.RunSync(func() { /* ordinary JS call */ })
//	loop.Stop()
package eventloop

import "runtime"

// EventLoop serialises all VM operations onto a single goroutine and calls a
// configurable checkpoint function after every task (used to drain microtask
// queues so async/await and Promise continuations run between calls).
type EventLoop struct {
	taskCh     chan func()
	stopCh     chan struct{}
	checkpoint func()
	lockOS     bool
}

// New creates and starts an EventLoop. If lockOS is true the loop goroutine is
// pinned to its OS thread via runtime.LockOSThread (required by V8).
func New(lockOS bool) *EventLoop {
	el := &EventLoop{
		taskCh:     make(chan func(), 256),
		stopCh:     make(chan struct{}),
		checkpoint: func() {},
		lockOS:     lockOS,
	}
	go el.run()
	return el
}

// SetCheckpoint replaces the post-task checkpoint. Must be called from inside a
// RunSync block (i.e. on the event loop goroutine) once the VM exists.
func (el *EventLoop) SetCheckpoint(fn func()) {
	el.checkpoint = fn
}

func (el *EventLoop) run() {
	if el.lockOS {
		runtime.LockOSThread()
	}
	for {
		select {
		case task := <-el.taskCh:
			task()
			el.checkpoint()
		case <-el.stopCh:
			return
		}
	}
}

// RunSync submits fn to the event loop goroutine and blocks until fn and the
// subsequent checkpoint both complete.
func (el *EventLoop) RunSync(fn func()) {
	done := make(chan struct{})
	el.taskCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// RunAsync queues fn on the event loop without blocking the caller.
// Used by bridge goroutines to schedule callbacks back onto the JS thread.
func (el *EventLoop) RunAsync(fn func()) {
	el.taskCh <- fn
}

// Stop terminates the event loop goroutine.
func (el *EventLoop) Stop() {
	close(el.stopCh)
}
