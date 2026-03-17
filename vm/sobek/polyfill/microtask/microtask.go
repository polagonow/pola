// Package microtask sets up the custom microtask queue used by React's Flight
// encoder inside the Sobek VM.
//
// Globals registered:
//   - __microtaskQueue__  — a real sobek Array (JS code can also call .push())
//   - queueMicrotask(fn) — appends fn to the queue
//   - __drainMicrotasks__() — splices and calls all queued fns; called
//     explicitly inside __pullStream__ to force Flight work to complete
//     before chunks are collected in the same tick.
package microtask

import (
	"strconv"

	sobeklib "github.com/grafana/sobek"
)

// Enable installs the microtask queue globals onto rt.
// Must be called before messagechannel and readablestream.
func Enable(rt *sobeklib.Runtime) {
	queue := rt.NewArray()
	rt.Set("__microtaskQueue__", queue) //nolint:errcheck

	pushFn, _ := sobeklib.AssertFunction(queue.Get("push"))

	rt.Set("queueMicrotask", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		fn := call.Argument(0)
		pushFn(queue, fn) //nolint:errcheck
		return sobeklib.Undefined()
	})

	spliceFn, _ := sobeklib.AssertFunction(queue.Get("splice"))

	rt.Set("__drainMicrotasks__", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		safety := 0
		for queue.Get("length").ToInteger() > 0 && safety < 5000 {
			safety++
			batchVal, err := spliceFn(queue, rt.ToValue(0))
			if err != nil {
				break
			}
			batch := batchVal.(*sobeklib.Object)
			length := batch.Get("length").ToInteger()
			for i := int64(0); i < length; i++ {
				fn := batch.Get(strconv.FormatInt(i, 10))
				if callable, ok := sobeklib.AssertFunction(fn); ok {
					func() {
						defer func() { recover() }()   //nolint:errcheck
						callable(sobeklib.Undefined()) //nolint:errcheck
					}()
				}
			}
		}
		return sobeklib.Undefined()
	})
}
