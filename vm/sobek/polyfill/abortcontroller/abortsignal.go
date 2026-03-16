package abortcontroller

import (
	"strconv"

	sobeklib "github.com/grafana/sobek"
)

// buildSignalProto returns the prototype shared by all AbortSignal instances.
func buildSignalProto(rt *sobeklib.Runtime) *sobeklib.Object {
	proto := rt.NewObject()

	proto.Set("addEventListener", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		if call.Argument(0).String() != "abort" {
			return sobeklib.Undefined()
		}
		fn := call.Argument(1)
		this := call.This.ToObject(rt)
		listeners := this.Get("_listeners").ToObject(rt)
		pushFn, _ := sobeklib.AssertFunction(listeners.Get("push"))
		pushFn(listeners, fn) //nolint:errcheck
		return sobeklib.Undefined()
	})

	proto.Set("removeEventListener", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		if call.Argument(0).String() != "abort" {
			return sobeklib.Undefined()
		}
		target := call.Argument(1)
		this := call.This.ToObject(rt)
		listeners := this.Get("_listeners").ToObject(rt)
		length := listeners.Get("length").ToInteger()

		newArr := rt.NewArray()
		pushFn, _ := sobeklib.AssertFunction(newArr.Get("push"))
		for i := int64(0); i < length; i++ {
			item := listeners.Get(strconv.FormatInt(i, 10))
			if item != target {
				pushFn(newArr, item) //nolint:errcheck
			}
		}
		this.Set("_listeners", newArr) //nolint:errcheck
		return sobeklib.Undefined()
	})

	proto.Set("throwIfAborted", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		if this.Get("aborted").ToBoolean() {
			panic(rt.ToValue(this.Get("reason")))
		}
		return sobeklib.Undefined()
	})

	return proto
}
