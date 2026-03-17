// Package abortcontroller provides AbortController and AbortSignal for the
// Sobek VM.
//
// react-server-dom-webpack/server.browser uses AbortController for request
// cancellation. This covers:
//   - new AbortController() with .signal
//   - ac.abort(reason?) — sets signal.aborted, fires listeners synchronously
//   - signal.addEventListener / removeEventListener / throwIfAborted
package abortcontroller

import (
	"strconv"

	sobeklib "github.com/grafana/sobek"
)

// Enable installs AbortSignal and AbortController as globals onto rt.
func Enable(rt *sobeklib.Runtime) {
	signalProto := buildSignalProto(rt)

	signalCtor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		call.This.Set("aborted", false)               //nolint:errcheck
		call.This.Set("reason", sobeklib.Undefined()) //nolint:errcheck
		call.This.Set("_listeners", rt.NewArray())    //nolint:errcheck
		call.This.SetPrototype(signalProto)           //nolint:errcheck
		return nil
	}).(*sobeklib.Object)
	signalCtor.Set("prototype", signalProto) //nolint:errcheck
	rt.Set("AbortSignal", signalCtor)        //nolint:errcheck

	acProto := rt.NewObject()

	acProto.Set("abort", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		signal := this.Get("signal").ToObject(rt)
		if signal.Get("aborted").ToBoolean() {
			return sobeklib.Undefined()
		}
		signal.Set("aborted", true) //nolint:errcheck

		reason := call.Argument(0)
		if sobeklib.IsUndefined(reason) {
			errCtor, ok := sobeklib.AssertConstructor(rt.Get("Error"))
			if ok {
				r, err := errCtor(nil, rt.ToValue("AbortError"))
				if err == nil {
					reason = r
				}
			}
			if sobeklib.IsUndefined(reason) {
				reason = rt.ToValue("AbortError")
			}
		}
		signal.Set("reason", reason) //nolint:errcheck

		listeners := signal.Get("_listeners").ToObject(rt)
		length := listeners.Get("length").ToInteger()
		evt := rt.NewObject()
		evt.Set("type", "abort")  //nolint:errcheck
		evt.Set("target", signal) //nolint:errcheck
		for i := int64(0); i < length; i++ {
			fn := listeners.Get(strconv.FormatInt(i, 10))
			if callable, ok := sobeklib.AssertFunction(fn); ok {
				func() {
					defer func() { _ = recover() }()
					callable(sobeklib.Undefined(), evt) //nolint:errcheck
				}()
			}
		}
		return sobeklib.Undefined()
	})

	acCtor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		signalNew, ok := sobeklib.AssertConstructor(rt.Get("AbortSignal"))
		if !ok {
			panic(rt.NewTypeError("AbortSignal is not a constructor"))
		}
		sig, err := signalNew(call.This)
		if err != nil {
			panic(rt.NewTypeError("AbortController: failed to create signal: %v", err))
		}
		call.This.Set("signal", sig)    //nolint:errcheck
		call.This.SetPrototype(acProto) //nolint:errcheck
		return nil
	}).(*sobeklib.Object)
	acCtor.Set("prototype", acProto)  //nolint:errcheck
	rt.Set("AbortController", acCtor) //nolint:errcheck
}
