// Package textencoding provides TextEncoder and TextDecoder for the Sobek VM.
package textencoding

import (
	"strconv"

	sobeklib "github.com/grafana/sobek"
)

// Enable installs TextEncoder and TextDecoder as globals onto rt.
func Enable(rt *sobeklib.Runtime) {
	enableEncoder(rt)
	enableDecoder(rt)
}

func enableEncoder(rt *sobeklib.Runtime) {
	proto := rt.NewObject()

	proto.Set("encode", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		s := call.Argument(0).String()
		data := []byte(s) // Go string is always UTF-8
		ab := rt.NewArrayBuffer(data)
		uint8Ctor, ok := sobeklib.AssertConstructor(rt.Get("Uint8Array"))
		if !ok {
			panic(rt.NewTypeError("Uint8Array constructor not available"))
		}
		obj, err := uint8Ctor(nil, rt.ToValue(ab))
		if err != nil {
			panic(rt.NewTypeError("TextEncoder.encode: %v", err))
		}
		return obj
	})

	proto.Set("encodeInto", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		s := call.Argument(0).String()
		dest := call.Argument(1).ToObject(rt)
		data := []byte(s)
		destLen := dest.Get("length").ToInteger()
		written := int64(len(data))
		if written > destLen {
			written = destLen
		}
		for i := int64(0); i < written; i++ {
			dest.Set(strconv.FormatInt(i, 10), rt.ToValue(data[i])) //nolint:errcheck
		}
		result := rt.NewObject()
		result.Set("read", rt.ToValue(int64(len([]rune(s)))))    //nolint:errcheck
		result.Set("written", rt.ToValue(written))               //nolint:errcheck
		return result
	})

	ctor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		call.This.Set("encoding", "utf-8") //nolint:errcheck
		call.This.SetPrototype(proto)      //nolint:errcheck
		return nil
	}).(*sobeklib.Object)
	ctor.Set("prototype", proto) //nolint:errcheck
	rt.Set("TextEncoder", ctor)  //nolint:errcheck
}

func enableDecoder(rt *sobeklib.Runtime) {
	proto := rt.NewObject()

	proto.Set("decode", func(call sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
		arg := call.Argument(0)
		if sobeklib.IsUndefined(arg) || sobeklib.IsNull(arg) {
			return rt.ToValue("")
		}

		// Try exporting as []byte first (covers Uint8Array and ArrayBuffer).
		var data []byte
		if err := rt.ExportTo(arg, &data); err == nil {
			return rt.ToValue(string(data))
		}

		// Fallback: treat as array-like, read numeric indices.
		obj := arg.ToObject(rt)
		length := obj.Get("length").ToInteger()
		data = make([]byte, length)
		for i := int64(0); i < length; i++ {
			data[i] = byte(obj.Get(strconv.FormatInt(i, 10)).ToInteger())
		}
		return rt.ToValue(string(data))
	})

	ctor := rt.ToValue(func(call sobeklib.ConstructorCall) *sobeklib.Object {
		enc := call.Argument(0)
		if sobeklib.IsUndefined(enc) {
			call.This.Set("encoding", "utf-8") //nolint:errcheck
		} else {
			call.This.Set("encoding", enc.String()) //nolint:errcheck
		}
		call.This.SetPrototype(proto) //nolint:errcheck
		return nil
	}).(*sobeklib.Object)
	ctor.Set("prototype", proto) //nolint:errcheck
	rt.Set("TextDecoder", ctor)  //nolint:errcheck
}
