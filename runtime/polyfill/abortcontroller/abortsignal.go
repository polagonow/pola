package abortcontroller

import (
	"strconv"

	"github.com/dop251/goja"
)

// buildSignalProto returns the prototype shared by all AbortSignal instances.
func buildSignalProto(rt *goja.Runtime) *goja.Object {
	proto := rt.NewObject()

	proto.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		if call.Argument(0).String() != "abort" {
			return goja.Undefined()
		}
		fn := call.Argument(1)
		this := call.This.ToObject(rt)
		listeners := this.Get("_listeners").ToObject(rt)
		pushFn, _ := goja.AssertFunction(listeners.Get("push"))
		pushFn(listeners, fn) //nolint:errcheck
		return goja.Undefined()
	})

	proto.Set("removeEventListener", func(call goja.FunctionCall) goja.Value {
		if call.Argument(0).String() != "abort" {
			return goja.Undefined()
		}
		target := call.Argument(1)
		this := call.This.ToObject(rt)
		listeners := this.Get("_listeners").ToObject(rt)
		length := listeners.Get("length").ToInteger()

		newArr := rt.NewArray()
		pushFn, _ := goja.AssertFunction(newArr.Get("push"))
		for i := int64(0); i < length; i++ {
			item := listeners.Get(strconv.FormatInt(i, 10))
			if item != target {
				pushFn(newArr, item) //nolint:errcheck
			}
		}
		this.Set("_listeners", newArr)
		return goja.Undefined()
	})

	proto.Set("throwIfAborted", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		if this.Get("aborted").ToBoolean() {
			panic(rt.ToValue(this.Get("reason")))
		}
		return goja.Undefined()
	})

	return proto
}
