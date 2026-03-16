// Package readablestream provides ReadableStream and __pullStream__ for the
// Sobek VM.
//
// __pullStream__ is the locked Go-bridge API called by render.go:
//
//	s._start() → __drainMicrotasks__() → s._pull() → __drainMicrotasks__()
//	→ splice chunks → return { chunks, done }
//
// Requires the microtask package to have been enabled first.
package readablestream

import sobeklib "github.com/grafana/sobek"

// Enable installs ReadableStream and __pullStream__ as globals onto rt.
func Enable(rt *sobeklib.Runtime) {
	controllerProto := buildControllerProto(rt)
	streamProto := buildStreamProto(rt)

	rsCtor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		controller := rt.NewObject()
		controller.SetPrototype(controllerProto)        //nolint:errcheck
		controller.Set("_chunks", rt.NewArray())        //nolint:errcheck
		controller.Set("_closed", rt.ToValue(false))    //nolint:errcheck
		controller.Set("_error", sobeklib.Null())       //nolint:errcheck

		src := call.Argument(0)
		if sobeklib.IsUndefined(src) || sobeklib.IsNull(src) {
			src = rt.NewObject()
		}
		call.This.Set("_controller", controller)        //nolint:errcheck
		call.This.Set("_src", src)                      //nolint:errcheck
		call.This.Set("_started", rt.ToValue(false))    //nolint:errcheck
		call.This.SetPrototype(streamProto)              //nolint:errcheck
		return nil
	}).(*sobeklib.Object)
	rt.Set("ReadableStream", rsCtor) //nolint:errcheck

	rt.Set("__pullStream__", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		s := call.Argument(0).ToObject(rt)

		callMethod(rt, s, "_start")

		if drain, ok := sobeklib.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(sobeklib.Undefined()) //nolint:errcheck
		}

		callMethod(rt, s, "_pull")

		if drain, ok := sobeklib.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(sobeklib.Undefined()) //nolint:errcheck
		}

		controller := s.Get("_controller").ToObject(rt)
		chunksArr := controller.Get("_chunks").ToObject(rt)
		spliceFn, _ := sobeklib.AssertFunction(chunksArr.Get("splice"))
		chunks, _ := spliceFn(chunksArr, rt.ToValue(0))

		closed := controller.Get("_closed").ToBoolean()
		chunksLen := chunks.ToObject(rt).Get("length").ToInteger()

		result := rt.NewObject()
		result.Set("chunks", chunks)                                     //nolint:errcheck
		result.Set("done", rt.ToValue(closed && chunksLen == 0))         //nolint:errcheck
		return result
	})
}

func buildControllerProto(rt *sobeklib.Runtime) *sobeklib.Object {
	proto := rt.NewObject()

	proto.Set("enqueue", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		if this.Get("_closed").ToBoolean() {
			return sobeklib.Undefined()
		}
		chunks := this.Get("_chunks").ToObject(rt)
		pushFn, _ := sobeklib.AssertFunction(chunks.Get("push"))
		pushFn(chunks, call.Argument(0)) //nolint:errcheck
		return sobeklib.Undefined()
	})

	proto.Set("close", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		call.This.ToObject(rt).Set("_closed", rt.ToValue(true)) //nolint:errcheck
		return sobeklib.Undefined()
	})

	proto.Set("error", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		this.Set("_error", call.Argument(0))     //nolint:errcheck
		this.Set("_closed", rt.ToValue(true))    //nolint:errcheck
		return sobeklib.Undefined()
	})

	proto.DefineAccessorProperty("byobRequest",
		rt.ToValue(func(call sobeklib.FunctionCall) sobeklib.Value { return sobeklib.Null() }),
		sobeklib.Undefined(),
		sobeklib.FLAG_FALSE, sobeklib.FLAG_TRUE,
	) //nolint:errcheck

	proto.DefineAccessorProperty("desiredSize",
		rt.ToValue(func(call sobeklib.FunctionCall) sobeklib.Value {
			this := call.This.ToObject(rt)
			if this.Get("_closed").ToBoolean() {
				return rt.ToValue(0)
			}
			return rt.ToValue(1)
		}),
		sobeklib.Undefined(),
		sobeklib.FLAG_FALSE, sobeklib.FLAG_TRUE,
	) //nolint:errcheck

	return proto
}

func buildStreamProto(rt *sobeklib.Runtime) *sobeklib.Object {
	proto := rt.NewObject()

	proto.Set("_start", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		if this.Get("_started").ToBoolean() {
			return sobeklib.Undefined()
		}
		this.Set("_started", rt.ToValue(true)) //nolint:errcheck
		src := this.Get("_src").ToObject(rt)
		if startFn, ok := sobeklib.AssertFunction(src.Get("start")); ok {
			startFn(src, this.Get("_controller")) //nolint:errcheck
		}
		return sobeklib.Undefined()
	})

	proto.Set("_pull", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		this := call.This.ToObject(rt)
		src := this.Get("_src").ToObject(rt)
		if pullFn, ok := sobeklib.AssertFunction(src.Get("pull")); ok {
			pullFn(src, this.Get("_controller")) //nolint:errcheck
		}
		return sobeklib.Undefined()
	})

	return proto
}

func callMethod(rt *sobeklib.Runtime, obj *sobeklib.Object, name string) {
	fn, ok := sobeklib.AssertFunction(obj.Get(name))
	if !ok {
		return
	}
	fn(obj) //nolint:errcheck
}
