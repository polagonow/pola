package v8go

import (
	"encoding/json"
	"fmt"

	v8 "rogchap.com/v8go"
)

// goToV8Value converts a Go value to a v8go Value.
// Primitives (nil, bool, string, int*, float*) are converted directly.
// Complex types (map, slice, struct) are JSON-encoded and evaluated in JS.
func goToV8Value(iso *v8.Isolate, ctx *v8.Context, v any) (*v8.Value, error) {
	switch t := v.(type) {
	case nil:
		return v8.Null(iso), nil
	case bool:
		return v8.NewValue(iso, t)
	case string:
		return v8.NewValue(iso, t)
	case int:
		return v8.NewValue(iso, int32(t))
	case int32:
		return v8.NewValue(iso, t)
	case int64:
		return v8.NewValue(iso, t)
	case uint32:
		return v8.NewValue(iso, t)
	case uint64:
		return v8.NewValue(iso, t)
	case float32:
		return v8.NewValue(iso, float64(t))
	case float64:
		return v8.NewValue(iso, t)
	default:
		// JSON-encode complex values (maps, slices, structs) and parse them in V8.
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("v8 value: json marshal: %w", err)
		}
		return ctx.RunScript("("+string(data)+")", "value.js")
	}
}

// exportV8Value converts a single *v8.Value to a plain Go interface{} value.
// Primitives are returned as their native Go types; objects/arrays become their
// JSON string (compatible with how bridge functions use fmt.Sprintf("%v", arg)).
func exportV8Value(v *v8.Value) any {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	if v.IsBoolean() {
		return v.Boolean()
	}
	if v.IsNumber() {
		n := v.Number()
		if n == float64(int64(n)) {
			return int64(n)
		}
		return n
	}
	if v.IsString() {
		return v.String()
	}
	// For arrays, objects, etc. fall back to the JS string representation.
	return v.String()
}

// exportV8Args converts v8go function arguments to plain Go interface{} values.
func exportV8Args(args []*v8.Value) []any {
	out := make([]any, len(args))
	for i, v := range args {
		out[i] = exportV8Value(v)
	}
	return out
}

// readUint8Array reads all bytes from a JavaScript Uint8Array object.
func readUint8Array(obj *v8.Object) []byte {
	lenVal, err := obj.Get("length")
	if err != nil {
		return nil
	}
	n := int(lenVal.Integer())
	data := make([]byte, n)
	for i := 0; i < n; i++ {
		v, err := obj.GetIdx(uint32(i))
		if err != nil {
			continue
		}
		data[i] = byte(v.Integer())
	}
	return data
}
