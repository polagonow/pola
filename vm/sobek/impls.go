package sobek

import (
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"
	"gojsx/framework/globals"

	sobeklib "github.com/grafana/sobek"
)

// ── StreamHandle ─────────────────────────────────────────────────────────

// StreamHandle implements framework.StreamHandle for Sobek-rendered streams.
type StreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *StreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return vm.run(func(rt *sobeklib.Runtime) error {
		return rt.Set(globals.RequestContext, rt.ToValue(ctx))
	})
}

// SetBridgeFunctions implements framework.VM.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	return vm.run(func(rt *sobeklib.Runtime) error {
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		for name, fn := range funcs {
			vm.jsi.Set(name, func(c sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
				args := exportArgs(c.Arguments)
				p, resolve, reject := rt.NewPromise()
				go func() {
					result, err := fn(args)
					vm.loop.RunAsync(func() {
						if err != nil {
							reject(rt.ToValue(err.Error())) //nolint:errcheck
						} else {
							resolve(rt.ToValue(result)) //nolint:errcheck
						}
					})
				}()
				return rt.ToValue(p)
			})
		}
		return nil
	})
}

// CallRenderFunction implements framework.VM.
func (vm *VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess, err := StartRender(vm, exportName, propsJSON)
	if err != nil {
		return &StreamHandle{}, err
	}
	return &StreamHandle{Sess: sess}, nil
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	sobekHandle, ok := handle.(*StreamHandle)
	if !ok {
		return false, fmt.Errorf("sobek DrainStream: expected *StreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, sobekHandle.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	return vm.run(func(rt *sobeklib.Runtime) error {
		rt.Set(globals.RequestContext, sobeklib.Undefined()) //nolint:errcheck
		rt.Set(globals.StreamHandle, sobeklib.Undefined())   //nolint:errcheck
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		return nil
	})
}

// ── VMPool ───────────────────────────────────────────────────────────────

// VMPool wraps the internal pool and implements framework.VMPool.
type VMPool struct {
	inner *vmPool
}

// NewVMPool creates a framework.VMPool backed by a pre-warmed Sobek pool.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	inner, err := newVMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &VMPool{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *VMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *VMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── VMFactory ────────────────────────────────────────────────────────────

// VMFactory implements framework.VMFactory for Sobek.
type VMFactory struct {
	prog *sobeklib.Program
}

// NewVMFactory compiles serverBundle and returns a factory ready to create VMs.
func NewVMFactory(serverBundle []byte) (*VMFactory, error) {
	prog, err := sobeklib.Compile("bundle.js", string(serverBundle), false)
	if err != nil {
		return nil, fmt.Errorf("sobek factory: compile: %w", err)
	}
	return &VMFactory{prog: prog}, nil
}

// New implements framework.VMFactory.
func (f *VMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newVM(f.prog, bridge)
}
