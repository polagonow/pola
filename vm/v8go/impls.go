package v8go

import (
	"encoding/json"
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"

	v8 "rogchap.com/v8go"
)

// ── V8StreamHandle ─────────────────────────────────────────────────────────

// V8StreamHandle implements framework.StreamHandle for v8go-rendered streams.
type V8StreamHandle struct {
	Sess *V8RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *V8StreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *V8VM ──────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *V8VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	data, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("v8: marshal request context: %w", err)
	}
	return vm.run(func() error {
		_, err := vm.ctx.RunScript("globalThis.__REQUEST__ = "+string(data)+";", "request.js")
		return err
	})
}

// SetBridgeFunctions implements framework.VM.
func (vm *V8VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	return vm.run(func() error {
		// Clear previously registered JSI keys.
		for _, key := range vm.jsiKeys {
			vm.ctx.RunScript(fmt.Sprintf("delete __JSI__[%q];", key), "jsi_clear.js") //nolint:errcheck
		}
		vm.jsiKeys = vm.jsiKeys[:0]

		if len(funcs) == 0 {
			return nil
		}

		jsiVal, err := vm.ctx.Global().Get("__JSI__")
		if err != nil {
			return fmt.Errorf("v8: get __JSI__: %w", err)
		}
		jsiObj := jsiVal.Object()

		for name, fn := range funcs {
			ft := v8.NewFunctionTemplate(vm.iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
				args := exportV8Args(info.Args())
				resolver, err := v8.NewPromiseResolver(info.Context())
				if err != nil {
					return nil
				}
				go func() {
					result, err := fn(args)
					vm.loop.RunAsync(func() {
						if err != nil {
							errVal, _ := v8.NewValue(vm.iso, err.Error())
							resolver.Reject(errVal) //nolint:errcheck
						} else {
							val, convErr := goToV8Value(vm.iso, vm.ctx, result)
							if convErr != nil {
								errVal, _ := v8.NewValue(vm.iso, convErr.Error())
								resolver.Reject(errVal) //nolint:errcheck
							} else {
								resolver.Resolve(val) //nolint:errcheck
							}
						}
					})
				}()
				return resolver.GetPromise().Value
			})
			jsiObj.Set(name, ft.GetFunction(vm.ctx)) //nolint:errcheck
			vm.jsiKeys = append(vm.jsiKeys, name)
		}
		return nil
	})
}

// CallRenderFunction implements framework.VM.
func (vm *V8VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess, err := StartRender(vm, exportName, propsJSON)
	if err != nil {
		return &V8StreamHandle{}, err
	}
	return &V8StreamHandle{Sess: sess}, nil
}

// ClearState implements framework.VM.
func (vm *V8VM) ClearState() error {
	return vm.run(func() error {
		for _, key := range vm.jsiKeys {
			vm.ctx.RunScript(fmt.Sprintf("delete __JSI__[%q];", key), "jsi_clear.js") //nolint:errcheck
		}
		vm.jsiKeys = vm.jsiKeys[:0]
		vm.ctx.RunScript( //nolint:errcheck
			"globalThis.__REQUEST__ = undefined; globalThis.__gojsx_stream__ = undefined;", "clear.js")
		return nil
	})
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *V8VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	v8Handle, ok := handle.(*V8StreamHandle)
	if !ok {
		return false, fmt.Errorf("v8 DrainStream: expected *V8StreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, v8Handle.Sess)
}

// ── V8VMFactory ────────────────────────────────────────────────────────────

// V8VMFactory implements framework.VMFactory for v8go.
type V8VMFactory struct {
	source string
}

// NewV8VMFactory creates a factory from the compiled server bundle bytes.
func NewV8VMFactory(serverBundle []byte) (*V8VMFactory, error) {
	return &V8VMFactory{source: string(serverBundle)}, nil
}

// New implements framework.VMFactory.
func (f *V8VMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newV8VM(f.source, bridge)
}

// ── V8VMPoolWrapper ────────────────────────────────────────────────────────

// V8VMPoolWrapper wraps *V8VMPool and implements framework.VMPool.
type V8VMPoolWrapper struct {
	inner *V8VMPool
}

// NewV8VMPoolWrapper creates a framework.VMPool backed by a pre-warmed v8go pool.
func NewV8VMPoolWrapper(serverBundle string, bridge contract.BridgeConfig) (*V8VMPoolWrapper, error) {
	inner, err := NewV8VMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &V8VMPoolWrapper{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *V8VMPoolWrapper) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *V8VMPoolWrapper) Release(vm framework.VM) { p.inner.Release(vm.(*V8VM)) }
