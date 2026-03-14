// Package message_channel provides MessageChannel for the Goja VM.
//
// Requires the microtask package to have been enabled first (reads
// __microtaskQueue__ from the runtime at call time).
package messagechannel

import (
	"github.com/dop251/goja"
)

// Enable installs MessageChannel as a global constructor onto rt.
func Enable(rt *goja.Runtime) {
	portProto := rt.NewObject()

	portProto.Set("postMessage", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		data := call.Argument(0)
		partner := this.Get("_partner")
		if goja.IsUndefined(partner) || goja.IsNull(partner) {
			return goja.Undefined()
		}
		partnerObj := partner.ToObject(rt)

		queueObj := rt.Get("__microtaskQueue__").ToObject(rt)
		pushFn, ok := goja.AssertFunction(queueObj.Get("push"))
		if !ok {
			return goja.Undefined()
		}
		handler := rt.ToValue(func(call goja.FunctionCall) goja.Value {
			onmessage := partnerObj.Get("onmessage")
			if callable, ok := goja.AssertFunction(onmessage); ok {
				evt := rt.NewObject()
				evt.Set("data", data)
				callable(partnerObj, evt) //nolint:errcheck
			}
			return goja.Undefined()
		})
		pushFn(queueObj, handler) //nolint:errcheck
		return goja.Undefined()
	})

	portProto.Set("close", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	mcCtor := rt.ToValue(func(call goja.ConstructorCall) *goja.Object {
		port1 := rt.NewObject()
		port1.SetPrototype(portProto)
		port1.Set("onmessage", goja.Null())

		port2 := rt.NewObject()
		port2.SetPrototype(portProto)
		port2.Set("onmessage", goja.Null())

		port1.Set("_partner", port2)
		port2.Set("_partner", port1)

		call.This.Set("port1", port1)
		call.This.Set("port2", port2)
		return nil
	}).(*goja.Object)

	rt.Set("MessageChannel", mcCtor)
}
