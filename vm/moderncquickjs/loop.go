package moderncquickjs

// eventLoop serialises all modernc.org/quickjs operations onto a single goroutine.
// modernc.org/quickjs is not goroutine-safe; all VM access must go through RunSync or RunAsync.
//
// After every task the loop calls the checkpoint function. The checkpoint
// evals __drainMicrotasks__() to flush our custom microtask queue (React Flight
// uses queueMicrotask, not native Promises, so this is the relevant drain point).
type eventLoop struct {
	taskCh     chan func()
	stopCh     chan struct{}
	checkpoint func()
}

func newEventLoop() *eventLoop {
	el := &eventLoop{
		taskCh:     make(chan func(), 256),
		stopCh:     make(chan struct{}),
		checkpoint: func() {}, // replaced once VM is ready via setCheckpoint
	}
	go el.run()
	return el
}

// setCheckpoint wires the microtask drain function once the VM exists.
// Must be called from inside a RunSync block (i.e. on the event loop goroutine).
func (el *eventLoop) setCheckpoint(fn func()) {
	el.checkpoint = fn
}

func (el *eventLoop) run() {
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
// subsequent microtask checkpoint both complete.
func (el *eventLoop) RunSync(fn func()) {
	done := make(chan struct{})
	el.taskCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// RunAsync queues fn on the event loop without blocking the caller.
// Used by bridge goroutines to schedule callbacks back onto the JS thread.
func (el *eventLoop) RunAsync(fn func()) {
	el.taskCh <- fn
}

// Stop terminates the event loop goroutine.
func (el *eventLoop) Stop() {
	close(el.stopCh)
}
