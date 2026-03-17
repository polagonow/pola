package goja

import (
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"
	polyfill "gojsx/vm/goja/polyfill"

	gojalib "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// ── StreamHandle ──────────────────────────────────────────────────────────

// StreamHandle implements framework.StreamHandle for Goja-rendered streams.
type StreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *StreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetBridgeFunctions implements framework.VM.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	return vm.SetJSI(funcs)
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
	gojaHandle, ok := handle.(*StreamHandle)
	if !ok {
		return false, fmt.Errorf("goja DrainStream: expected *StreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, gojaHandle.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	return vm.run(func(rt *gojalib.Runtime) error {
		rt.Set("__REQUEST__", gojalib.Undefined())      //nolint:errcheck
		rt.Set("__gojsx_stream__", gojalib.Undefined()) //nolint:errcheck
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		return nil
	})
}

// ── VMPool ────────────────────────────────────────────────────────────────

// VMPool wraps the internal pool and implements framework.VMPool.
type VMPool struct {
	inner *vmPool
}

// NewVMPool creates a framework.VMPool backed by a pre-warmed Goja pool.
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

// Bridge returns the bridge config this pool was created with.
func (p *VMPool) Bridge() contract.BridgeConfig { return p.inner.Bridge() }

// ── VMFactory ─────────────────────────────────────────────────────────────

// VMFactory implements framework.VMFactory for Goja.
type VMFactory struct {
	prog      *gojalib.Program
	polyfills framework.PolyfillRegistry
}

// NewVMFactory compiles serverBundle and returns a factory ready to create VMs.
// Polyfills are managed internally using the default GojaPolyfillRegistry.
func NewVMFactory(serverBundle []byte) (*VMFactory, error) {
	prog, err := gojalib.Compile("bundle.js", string(serverBundle), false)
	if err != nil {
		return nil, fmt.Errorf("goja factory: compile: %w", err)
	}
	return &VMFactory{prog: prog, polyfills: &polyfill.GojaPolyfillRegistry{}}, nil
}

// New implements framework.VMFactory.
func (f *VMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false))

	vm := &VM{loop: loop, bridge: bridge}
	err := vm.run(func(rt *gojalib.Runtime) error {
		vm.rt = rt
		vm.jsi = rt.NewObject()
		rt.Set("__JSI__", vm.jsi)               //nolint:errcheck
		rt.Set("global", rt.GlobalObject())     //nolint:errcheck
		rt.Set("globalThis", rt.GlobalObject()) //nolint:errcheck
		rt.Set("console", map[string]any{       //nolint:errcheck
			"log":   func(c gojalib.FunctionCall) gojalib.Value { return logConsole("LOG", c) },
			"warn":  func(c gojalib.FunctionCall) gojalib.Value { return logConsole("WARN", c) },
			"error": func(c gojalib.FunctionCall) gojalib.Value { return logConsole("ERR", c) },
			"info":  func(c gojalib.FunctionCall) gojalib.Value { return logConsole("INFO", c) },
		})
		rt.Set("process", map[string]any{ //nolint:errcheck
			"env": map[string]any{"NODE_ENV": "production"},
		})
		rt.Set("performance", map[string]any{ //nolint:errcheck
			"now": func(c gojalib.FunctionCall) gojalib.Value { return rt.ToValue(0) },
		})

		for name, fn := range bridge.Globals {
			rt.Set(name, func(c gojalib.FunctionCall) gojalib.Value { //nolint:errcheck
				result, err := fn(exportArgs(c.Arguments))
				if err != nil {
					panic(rt.ToValue(err.Error()))
				}
				return rt.ToValue(result)
			})
		}

		ctx := &polyfill.GojaVMInitContext{Rt: rt}
		if err := f.polyfills.Install(ctx); err != nil {
			return fmt.Errorf("goja factory: polyfills: %w", err)
		}

		if _, err := rt.RunProgram(f.prog); err != nil {
			return fmt.Errorf("goja factory: run program: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}
