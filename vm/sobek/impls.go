package sobek

import (
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"

	sobeklib "github.com/grafana/sobek"
)

// ── SobekStreamHandle ─────────────────────────────────────────────────────────

// SobekStreamHandle implements framework.StreamHandle for Sobek-rendered streams.
type SobekStreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *SobekStreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return vm.run(func(rt *sobeklib.Runtime) error {
		return rt.Set("__REQUEST__", rt.ToValue(ctx))
	})
}

// SetBridgeFunctions implements framework.VM.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	return vm.run(func(rt *sobeklib.Runtime) error {
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		for name, fn := range funcs {
			fn := fn
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
		return &SobekStreamHandle{}, err
	}
	return &SobekStreamHandle{Sess: sess}, nil
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	sobekHandle, ok := handle.(*SobekStreamHandle)
	if !ok {
		return false, fmt.Errorf("sobek DrainStream: expected *SobekStreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, sobekHandle.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	return vm.run(func(rt *sobeklib.Runtime) error {
		rt.Set("__REQUEST__", sobeklib.Undefined())        //nolint:errcheck
		rt.Set("__gojsx_stream__", sobeklib.Undefined())   //nolint:errcheck
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		return nil
	})
}

// ── SobekVMPool ───────────────────────────────────────────────────────────────

// SobekVMPool wraps *VMPool and implements framework.VMPool.
type SobekVMPool struct {
	inner *VMPool
}

// NewSobekVMPool creates a framework.VMPool backed by a pre-warmed Sobek pool.
func NewSobekVMPool(serverBundle string, bridge contract.BridgeConfig) (*SobekVMPool, error) {
	inner, err := NewVMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &SobekVMPool{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *SobekVMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *SobekVMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── SobekVMFactory ────────────────────────────────────────────────────────────

// SobekVMFactory implements framework.VMFactory for Sobek.
type SobekVMFactory struct {
	prog *sobeklib.Program
}

// NewSobekVMFactory compiles serverBundle and returns a factory ready to create VMs.
func NewSobekVMFactory(serverBundle []byte) (*SobekVMFactory, error) {
	prog, err := sobeklib.Compile("bundle.js", string(serverBundle), false)
	if err != nil {
		return nil, fmt.Errorf("sobek factory: compile: %w", err)
	}
	return &SobekVMFactory{prog: prog}, nil
}

// New implements framework.VMFactory.
func (f *SobekVMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newVM(f.prog, bridge)
}
