// Package microtask sets up the custom microtask queue used by React's Flight
// encoder inside the Goja VM.
//
// Globals registered:
//   - __microtaskQueue__  — a real goja Array (JS code can also call .push())
//   - queueMicrotask(fn) — appends fn to the queue
//   - __drainMicrotasks__() — splices and calls all queued fns; called
//     explicitly inside __pullStream__ to force Flight work to complete
//     before chunks are collected in the same tick.
package microtask

import (
	"strconv"

	"github.com/dop251/goja"
)

// Enable installs the microtask queue globals onto rt.
// Must be called before message_channel and readable_stream.
func Enable(rt *goja.Runtime) {
	queue := rt.NewArray()
	rt.Set("__microtaskQueue__", queue)

	pushFn, _ := goja.AssertFunction(queue.Get("push"))

	rt.Set("queueMicrotask", func(call goja.FunctionCall) goja.Value {
		fn := call.Argument(0)
		pushFn(queue, fn) //nolint:errcheck
		return goja.Undefined()
	})

	spliceFn, _ := goja.AssertFunction(queue.Get("splice"))

	rt.Set("__drainMicrotasks__", func(call goja.FunctionCall) goja.Value {
		safety := 0
		for queue.Get("length").ToInteger() > 0 && safety < 5000 {
			safety++
			batchVal, err := spliceFn(queue, rt.ToValue(0))
			if err != nil {
				break
			}
			batch := batchVal.(*goja.Object)
			length := batch.Get("length").ToInteger()
			for i := int64(0); i < length; i++ {
				fn := batch.Get(strconv.FormatInt(i, 10))
				if callable, ok := goja.AssertFunction(fn); ok {
					func() {
						defer func() { recover() }() //nolint:errcheck — swallow, matching JS try/catch
						callable(goja.Undefined())    //nolint:errcheck
					}()
				}
			}
		}
		return goja.Undefined()
	})
}
