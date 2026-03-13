package runtime

import (
	"fmt"

	"github.com/dop251/goja"
)

// walkNode converts a Goja value (output of a Server Component) into a Go
// value suitable for JSON-encoding into a Flight payload chunk.
// Used by renderFallback when the real React server bundle is not loaded.
func walkNode(rt *goja.Runtime, val goja.Value, fw *FlightWriter, sched *SuspenseScheduler, manifest ClientManifest) (any, error) {
	if val == nil || goja.IsNull(val) || goja.IsUndefined(val) {
		return nil, nil
	}

	exported := val.Export()

	switch v := exported.(type) {
	case string:
		return v, nil
	case int64, float64, bool:
		return v, nil
	}

	obj := val.ToObject(rt)
	if obj == nil {
		return exported, nil
	}

	// Promise → async Suspense boundary
	if isPromise(rt, val) {
		boundaryID := sched.Register(
			map[string]string{"type": "loading"},
			func() (any, error) {
				resolved, err := resolvePromise(rt, val)
				if err != nil {
					return nil, err
				}
				return walkNode(rt, resolved, fw, sched, manifest)
			},
		)
		return map[string]any{
			"type":       "$Suspense",
			"boundaryID": boundaryID,
			"fallback":   map[string]any{"type": "div", "props": map[string]any{"children": "Loading..."}},
		}, nil
	}

	// Array → walk each child
	if isJSArray(rt, obj) {
		arr, ok := obj.Export().([]interface{})
		if !ok {
			return exported, nil
		}
		result := make([]any, len(arr))
		for i, item := range arr {
			node, err := walkNode(rt, rt.ToValue(item), fw, sched, manifest)
			if err != nil {
				return nil, fmt.Errorf("array[%d]: %w", i, err)
			}
			result[i] = node
		}
		return result, nil
	}

	// React element
	if isReactElement(obj) {
		return walkReactElement(rt, obj, fw, sched, manifest)
	}

	return exported, nil
}

func walkReactElement(rt *goja.Runtime, obj *goja.Object, fw *FlightWriter, sched *SuspenseScheduler, manifest ClientManifest) (any, error) {
	typeVal := obj.Get("type")
	propsVal := obj.Get("props")
	keyVal := obj.Get("key")

	var elemType any
	if typeVal != nil && !goja.IsUndefined(typeVal) && !goja.IsNull(typeVal) {
		if typeStr, ok := typeVal.Export().(string); ok {
			elemType = typeStr
		} else {
			typeObj := typeVal.ToObject(rt)
			if typeObj != nil {
				if marker := typeObj.Get("$$isClientReference"); marker != nil && marker.ToBoolean() {
					id := typeObj.Get("$$id").String()
					name := typeObj.Get("$$name").String()
					if name == "" {
						name = "default"
					}
					ref, ok := manifest[id]
					if !ok {
						ref = ClientRef{ID: id, Name: name, Chunks: []string{"main.js"}}
					}
					elemType = ref
				} else if fn, isFn := goja.AssertFunction(typeVal); isFn {
					propsArg := goja.Undefined()
					if propsVal != nil {
						propsArg = propsVal
					}
					result, err := fn(goja.Undefined(), propsArg)
					if err != nil {
						return nil, fmt.Errorf("server component: %w", err)
					}
					return walkNode(rt, result, fw, sched, manifest)
				} else {
					elemType = typeVal.String()
				}
			}
		}
	}

	var propsMap map[string]any
	if propsVal != nil && !goja.IsUndefined(propsVal) && !goja.IsNull(propsVal) {
		propsObj := propsVal.ToObject(rt)
		if propsObj != nil {
			propsMap = make(map[string]any)
			for _, key := range propsObj.Keys() {
				propVal := propsObj.Get(key)
				if key == "children" {
					children, err := walkNode(rt, propVal, fw, sched, manifest)
					if err != nil {
						return nil, fmt.Errorf("children: %w", err)
					}
					propsMap["children"] = children
				} else {
					propsMap[key] = propVal.Export()
				}
			}
		}
	}
	if propsMap == nil {
		propsMap = map[string]any{}
	}

	node := map[string]any{"type": elemType, "props": propsMap}
	if keyVal != nil && !goja.IsNull(keyVal) && !goja.IsUndefined(keyVal) {
		node["key"] = keyVal.Export()
	}
	return node, nil
}

func isPromise(rt *goja.Runtime, val goja.Value) bool {
	if val == nil || goja.IsNull(val) || goja.IsUndefined(val) {
		return false
	}
	obj := val.ToObject(rt)
	if obj == nil {
		return false
	}
	thenVal := obj.Get("then")
	if thenVal == nil || goja.IsUndefined(thenVal) {
		return false
	}
	_, ok := goja.AssertFunction(thenVal)
	return ok
}

func resolvePromise(rt *goja.Runtime, promiseVal goja.Value) (goja.Value, error) {
	var resolved goja.Value
	var rejected goja.Value
	done := make(chan struct{}, 1)

	thenFn, _ := goja.AssertFunction(promiseVal.ToObject(rt).Get("then"))
	catchFn, _ := goja.AssertFunction(promiseVal.ToObject(rt).Get("catch"))

	signal := func(ch chan struct{}) { select { case ch <- struct{}{}: default: } }

	resolveCb := rt.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			resolved = call.Arguments[0]
		}
		signal(done)
		return goja.Undefined()
	})
	rejectCb := rt.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			rejected = call.Arguments[0]
		}
		signal(done)
		return goja.Undefined()
	})

	if _, err := thenFn(goja.Undefined(), resolveCb); err != nil {
		return nil, fmt.Errorf("promise.then: %w", err)
	}
	if catchFn != nil {
		catchFn(goja.Undefined(), rejectCb) //nolint
	}

	select {
	case <-done:
	default:
		return nil, fmt.Errorf("promise still pending — requires event loop")
	}

	if rejected != nil {
		return nil, fmt.Errorf("promise rejected: %v", rejected.Export())
	}
	return resolved, nil
}

func isJSArray(rt *goja.Runtime, obj *goja.Object) bool {
	isArr, err := rt.RunString("Array.isArray")
	if err != nil {
		return false
	}
	fn, ok := goja.AssertFunction(isArr)
	if !ok {
		return false
	}
	result, err := fn(goja.Undefined(), obj)
	if err != nil {
		return false
	}
	b, ok := result.Export().(bool)
	return ok && b
}

func isReactElement(obj *goja.Object) bool {
	typeofVal := obj.Get("$$typeof")
	if typeofVal == nil || goja.IsUndefined(typeofVal) {
		return false
	}
	switch v := typeofVal.Export().(type) {
	case int64:
		return v == 0xeac7
	case float64:
		return v == 0xeac7
	}
	return true
}
