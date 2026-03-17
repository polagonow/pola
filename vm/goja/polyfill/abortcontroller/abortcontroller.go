// Package abort_controller provides AbortController and AbortSignal for the
// Goja VM.
//
// react-server-dom-webpack/server.browser uses AbortController for request
// cancellation. This covers:
//   - new AbortController() with .signal
//   - ac.abort(reason?) — sets signal.aborted, fires listeners synchronously
//   - signal.addEventListener / removeEventListener / throwIfAborted
package abortcontroller

import (
	"strconv"

	"github.com/dop251/goja"
)

// Enable installs AbortSignal and AbortController as globals onto rt.
func Enable(rt *goja.Runtime) {
	signalProto := buildSignalProto(rt)

	signalCtor := rt.ToValue(func(call goja.ConstructorCall) *goja.Object {
		call.This.Set("aborted", false)            //nolint:errcheck
		call.This.Set("reason", goja.Undefined())  //nolint:errcheck
		call.This.Set("_listeners", rt.NewArray()) //nolint:errcheck
		call.This.SetPrototype(signalProto)
		return nil
	}).(*goja.Object)
	signalCtor.Set("prototype", signalProto) //nolint:errcheck
	rt.Set("AbortSignal", signalCtor)        //nolint:errcheck

	acProto := rt.NewObject()

	acProto.Set("abort", func(call goja.FunctionCall) goja.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		signal := this.Get("signal").ToObject(rt)
		if signal.Get("aborted").ToBoolean() {
			return goja.Undefined()
		}
		signal.Set("aborted", true) //nolint:errcheck

		reason := call.Argument(0)
		if goja.IsUndefined(reason) {
			errCtor, ok := goja.AssertConstructor(rt.Get("Error"))
			if ok {
				r, err := errCtor(nil, rt.ToValue("AbortError"))
				if err == nil {
					reason = r
				}
			}
			if goja.IsUndefined(reason) {
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
			if callable, ok := goja.AssertFunction(fn); ok {
				func() {
					defer func() { _ = recover() }()
					callable(goja.Undefined(), evt) //nolint:errcheck
				}()
			}
		}
		return goja.Undefined()
	})

	acCtor := rt.ToValue(func(call goja.ConstructorCall) *goja.Object {
		signalNew, ok := goja.AssertConstructor(rt.Get("AbortSignal"))
		if !ok {
			panic(rt.NewTypeError("AbortSignal is not a constructor"))
		}
		sig, err := signalNew(call.This)
		if err != nil {
			panic(rt.NewTypeError("AbortController: failed to create signal: %v", err))
		}
		call.This.Set("signal", sig) //nolint:errcheck
		call.This.SetPrototype(acProto)
		return nil
	}).(*goja.Object)
	acCtor.Set("prototype", acProto)  //nolint:errcheck
	rt.Set("AbortController", acCtor) //nolint:errcheck
}
