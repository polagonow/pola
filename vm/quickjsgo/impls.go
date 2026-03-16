package quickjsgo

import (
	"fmt"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/framework"
	"gojsx/framework/contract"
)

// ── QuickJSStreamHandle ───────────────────────────────────────────────────────

// QuickJSStreamHandle implements framework.StreamHandle.
type QuickJSStreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *QuickJSStreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	var runErr error
	vm.run(func() {
		val, err := vm.ctx.Marshal(ctx)
		if err != nil {
			runErr = fmt.Errorf("quickjsgo: marshal request context: %w", err)
			return
		}
		g := vm.ctx.Globals()
		g.Set("__REQUEST__", val) //nolint:errcheck
		// Do NOT free val or g: Set transfers ownership of val, and g is a cached singleton.
	})
	return runErr
}

// SetBridgeFunctions implements framework.VM.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	vm.run(func() {
		// Clear existing keys.
		clearRet := vm.ctx.Eval(
			"Object.keys(__JSI__).forEach(function(k) { delete __JSI__[k]; })",
			quickjs.EvalFileName("clear_jsi.js"),
		)
		clearRet.Free()

		// Install new bridge functions.
		for name, fn := range funcs {
			name, fn := name, fn
			bridgeFn := vm.ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
				goArgs := exportArgs(args)

				// Save resolve/reject for the goroutine.
				var resolveFn, rejectFn func(*quickjs.Value)
				p := qCtx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
					resolveFn = resolve
					rejectFn = reject
				})

				go func() {
					result, err := fn(goArgs)
					qCtx.Schedule(func(innerCtx *quickjs.Context) {
						if err != nil {
							errVal := innerCtx.NewString(err.Error())
							rejectFn(errVal)
							errVal.Free()
						} else {
							val, marshalErr := innerCtx.Marshal(result)
							if marshalErr != nil {
								errVal := innerCtx.NewString(marshalErr.Error())
								rejectFn(errVal)
								errVal.Free()
								return
							}
							resolveFn(val)
							val.Free()
						}
					})
				}()

				return p
			})
			vm.jsi.Set(name, bridgeFn) //nolint:errcheck
			// Do NOT free bridgeFn: Set transfers ownership.
		}
	})
	return nil
}

// CallRenderFunction implements framework.VM.
func (vm *VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess, err := StartRender(vm, exportName, propsJSON)
	if err != nil {
		return &QuickJSStreamHandle{}, err
	}
	return &QuickJSStreamHandle{Sess: sess}, nil
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	qjsHandle, ok := handle.(*QuickJSStreamHandle)
	if !ok {
		return false, fmt.Errorf("quickjsgo DrainStream: expected *QuickJSStreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, qjsHandle.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	vm.run(func() {
		clearRet := vm.ctx.Eval(
			"__REQUEST__ = undefined; __gojsx_stream__ = undefined; __outputChunk__ = undefined; Object.keys(__JSI__).forEach(function(k) { delete __JSI__[k]; });",
			quickjs.EvalFileName("clear_state.js"),
		)
		clearRet.Free()
	})
	return nil
}

// ── QuickJSVMPool ─────────────────────────────────────────────────────────────

// QuickJSVMPool wraps *VMPool and implements framework.VMPool.
type QuickJSVMPool struct {
	inner *VMPool
}

// NewQuickJSVMPool creates a framework.VMPool backed by a pre-warmed pool.
func NewQuickJSVMPool(serverBundle string, bridge contract.BridgeConfig) (*QuickJSVMPool, error) {
	inner, err := NewVMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &QuickJSVMPool{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *QuickJSVMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *QuickJSVMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── QuickJSVMFactory ──────────────────────────────────────────────────────────

// QuickJSVMFactory implements framework.VMFactory for QuickJS.
type QuickJSVMFactory struct {
	src string
}

// NewQuickJSVMFactory stores serverBundle and returns a factory ready to create VMs.
func NewQuickJSVMFactory(serverBundle []byte) (*QuickJSVMFactory, error) {
	return &QuickJSVMFactory{src: string(serverBundle)}, nil
}

// New implements framework.VMFactory.
func (f *QuickJSVMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newVM(f.src, bridge)
}
