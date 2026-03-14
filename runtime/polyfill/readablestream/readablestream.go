// Package readable_stream provides ReadableStream and __pullStream__ for the
// Goja VM.
//
// __pullStream__ is the locked Go-bridge API called by render.go:
//
//	s._start() → __drainMicrotasks__() → s._pull() → __drainMicrotasks__()
//	→ splice chunks → return { chunks, done }
//
// Requires the microtask package to have been enabled first.
package readablestream

import (
	"github.com/dop251/goja"
)

// Enable installs ReadableStream and __pullStream__ as globals onto rt.
func Enable(rt *goja.Runtime) {
	controllerProto := buildControllerProto(rt)
	streamProto := buildStreamProto(rt)

	rsCtor := rt.ToValue(func(call goja.ConstructorCall) *goja.Object {
		controller := rt.NewObject()
		controller.SetPrototype(controllerProto)
		controller.Set("_chunks", rt.NewArray())
		controller.Set("_closed", rt.ToValue(false))
		controller.Set("_error", goja.Null())

		src := call.Argument(0)
		if goja.IsUndefined(src) || goja.IsNull(src) {
			src = rt.NewObject()
		}
		call.This.Set("_controller", controller)
		call.This.Set("_src", src)
		call.This.Set("_started", rt.ToValue(false))
		call.This.SetPrototype(streamProto)
		return nil
	}).(*goja.Object)
	rt.Set("ReadableStream", rsCtor)

	// __pullStream__ — global function, interface is locked (called by render.go).
	rt.Set("__pullStream__", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).ToObject(rt)

		callMethod(rt, s, "_start")

		if drain, ok := goja.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(goja.Undefined()) //nolint:errcheck
		}

		callMethod(rt, s, "_pull")

		if drain, ok := goja.AssertFunction(rt.Get("__drainMicrotasks__")); ok {
			drain(goja.Undefined()) //nolint:errcheck
		}

		controller := s.Get("_controller").ToObject(rt)
		chunksArr := controller.Get("_chunks").ToObject(rt)
		spliceFn, _ := goja.AssertFunction(chunksArr.Get("splice"))
		chunks, _ := spliceFn(chunksArr, rt.ToValue(0))

		closed := controller.Get("_closed").ToBoolean()
		chunksLen := chunks.ToObject(rt).Get("length").ToInteger()

		result := rt.NewObject()
		result.Set("chunks", chunks)
		result.Set("done", rt.ToValue(closed && chunksLen == 0))
		return result
	})
}

func buildControllerProto(rt *goja.Runtime) *goja.Object {
	proto := rt.NewObject()

	proto.Set("enqueue", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		if this.Get("_closed").ToBoolean() {
			return goja.Undefined()
		}
		chunks := this.Get("_chunks").ToObject(rt)
		pushFn, _ := goja.AssertFunction(chunks.Get("push"))
		pushFn(chunks, call.Argument(0)) //nolint:errcheck
		return goja.Undefined()
	})

	proto.Set("close", func(call goja.FunctionCall) goja.Value {
		call.This.ToObject(rt).Set("_closed", rt.ToValue(true))
		return goja.Undefined()
	})

	proto.Set("error", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		this.Set("_error", call.Argument(0))
		this.Set("_closed", rt.ToValue(true))
		return goja.Undefined()
	})

	proto.DefineAccessorProperty("byobRequest",
		rt.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Null() }),
		goja.Undefined(),
		goja.FLAG_FALSE, goja.FLAG_TRUE,
	)

	proto.DefineAccessorProperty("desiredSize",
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			this := call.This.ToObject(rt)
			if this.Get("_closed").ToBoolean() {
				return rt.ToValue(0)
			}
			return rt.ToValue(1)
		}),
		goja.Undefined(),
		goja.FLAG_FALSE, goja.FLAG_TRUE,
	)

	return proto
}

func buildStreamProto(rt *goja.Runtime) *goja.Object {
	proto := rt.NewObject()

	proto.Set("_start", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		if this.Get("_started").ToBoolean() {
			return goja.Undefined()
		}
		this.Set("_started", rt.ToValue(true))
		src := this.Get("_src").ToObject(rt)
		if startFn, ok := goja.AssertFunction(src.Get("start")); ok {
			startFn(src, this.Get("_controller")) //nolint:errcheck
		}
		return goja.Undefined()
	})

	proto.Set("_pull", func(call goja.FunctionCall) goja.Value {
		this := call.This.ToObject(rt)
		src := this.Get("_src").ToObject(rt)
		if pullFn, ok := goja.AssertFunction(src.Get("pull")); ok {
			pullFn(src, this.Get("_controller")) //nolint:errcheck
		}
		return goja.Undefined()
	})

	return proto
}

func callMethod(rt *goja.Runtime, obj *goja.Object, name string) {
	fn, ok := goja.AssertFunction(obj.Get(name))
	if !ok {
		return
	}
	fn(obj) //nolint:errcheck
}
