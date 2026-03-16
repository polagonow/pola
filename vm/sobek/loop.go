package sobek

import sobeklib "github.com/grafana/sobek"

// drainProg is a pre-compiled no-op that triggers Sobek's internal Promise
// microtask queue to drain. Called as a checkpoint after every loop task.
var drainProg, _ = sobeklib.Compile("drain.js", "0", false)

// eventLoop serialises all Sobek operations onto a single goroutine.
// Sobek is not goroutine-safe, so all rt access must go through RunSync or RunAsync.
type eventLoop struct {
	rt         *sobeklib.Runtime
	taskCh     chan func()
	stopCh     chan struct{}
	checkpoint func()
}

func newEventLoop() *eventLoop {
	el := &eventLoop{
		taskCh:     make(chan func(), 256),
		stopCh:     make(chan struct{}),
		checkpoint: func() {}, // replaced once rt is set
	}
	go el.run()
	return el
}

// setRuntime wires the runtime into the loop's checkpoint function.
// Must be called once, on the loop goroutine, before any tasks are queued.
func (el *eventLoop) setRuntime(rt *sobeklib.Runtime) {
	el.rt = rt
	el.checkpoint = func() {
		el.rt.RunProgram(drainProg) //nolint:errcheck
	}
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

// RunSync executes fn on the loop goroutine and blocks until fn and the
// subsequent microtask checkpoint both complete.
func (el *eventLoop) RunSync(fn func()) {
	done := make(chan struct{})
	el.taskCh <- func() {
		fn()
		close(done)
	}
	<-done
}

// RunAsync queues fn to run on the loop goroutine without blocking the caller.
// Used by bridge goroutines to resolve/reject Promises.
func (el *eventLoop) RunAsync(fn func()) {
	el.taskCh <- fn
}

// Stop shuts down the event loop goroutine.
func (el *eventLoop) Stop() {
	close(el.stopCh)
}
