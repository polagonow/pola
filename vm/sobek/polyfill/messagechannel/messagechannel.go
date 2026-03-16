// Package messagechannel provides MessageChannel for the Sobek VM.
//
// Requires the microtask package to have been enabled first (reads
// __microtaskQueue__ from the runtime at call time).
package messagechannel

import sobeklib "github.com/grafana/sobek"

// Enable installs MessageChannel as a global constructor onto rt.
func Enable(rt *sobeklib.Runtime) {
	portProto := rt.NewObject()

	portProto.Set("postMessage", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		data := call.Argument(0)
		partner := this.Get("_partner")
		if sobeklib.IsUndefined(partner) || sobeklib.IsNull(partner) {
			return sobeklib.Undefined()
		}
		partnerObj := partner.ToObject(rt)

		queueObj := rt.Get("__microtaskQueue__").ToObject(rt)
		pushFn, ok := sobeklib.AssertFunction(queueObj.Get("push"))
		if !ok {
			return sobeklib.Undefined()
		}
		handler := rt.ToValue(func(call sobeklib.FunctionCall) sobeklib.Value {
			onmessage := partnerObj.Get("onmessage")
			if callable, ok := sobeklib.AssertFunction(onmessage); ok {
				evt := rt.NewObject()
				evt.Set("data", data)           //nolint:errcheck
				callable(partnerObj, evt)        //nolint:errcheck
			}
			return sobeklib.Undefined()
		})
		pushFn(queueObj, handler) //nolint:errcheck
		return sobeklib.Undefined()
	})

	portProto.Set("close", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		return sobeklib.Undefined()
	})

	mcCtor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		port1 := rt.NewObject()
		port1.SetPrototype(portProto)      //nolint:errcheck
		port1.Set("onmessage", sobeklib.Null()) //nolint:errcheck

		port2 := rt.NewObject()
		port2.SetPrototype(portProto)      //nolint:errcheck
		port2.Set("onmessage", sobeklib.Null()) //nolint:errcheck

		port1.Set("_partner", port2) //nolint:errcheck
		port2.Set("_partner", port1) //nolint:errcheck

		call.This.Set("port1", port1) //nolint:errcheck
		call.This.Set("port2", port2) //nolint:errcheck
		return nil
	}).(*sobeklib.Object)

	rt.Set("MessageChannel", mcCtor) //nolint:errcheck
}
