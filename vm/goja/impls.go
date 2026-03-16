package goja

import (
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"
	polyfill "gojsx/vm/goja/polyfill"

	gojalib "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// ── GojaStreamHandle ──────────────────────────────────────────────────────────

// GojaStreamHandle implements framework.StreamHandle for Goja-rendered streams.
type GojaStreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *GojaStreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetBridgeFunctions implements framework.VM.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	return vm.SetJSI(funcs)
}

// CallRenderFunction implements framework.VM.
func (vm *VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess, err := StartRender(vm, exportName, propsJSON)
	if err != nil {
		return &GojaStreamHandle{}, err
	}
	return &GojaStreamHandle{Sess: sess}, nil
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	return vm.run(func(rt *gojalib.Runtime) error {
		rt.Set("__REQUEST__", gojalib.Undefined())
		rt.Set("__gojsx_stream__", gojalib.Undefined())
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		return nil
	})
}

// ── GojaVMPool ────────────────────────────────────────────────────────────────

// GojaVMPool wraps *VMPool and implements framework.VMPool.
type GojaVMPool struct {
	inner *VMPool
}

// NewGojaVMPool creates a framework.VMPool backed by a pre-warmed Goja pool.
func NewGojaVMPool(serverBundle string, bridge contract.BridgeConfig) (*GojaVMPool, error) {
	inner, err := NewVMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &GojaVMPool{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *GojaVMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *GojaVMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── GojaVMFactory ─────────────────────────────────────────────────────────────

// GojaVMFactory implements framework.VMFactory for Goja.
type GojaVMFactory struct {
	prog      *gojalib.Program
	polyfills framework.PolyfillRegistry
}

// NewGojaVMFactory compiles serverBundle and returns a factory ready to create VMs.
// Polyfills are managed internally using the default GojaPolyfillRegistry.
func NewGojaVMFactory(serverBundle []byte) (*GojaVMFactory, error) {
	prog, err := gojalib.Compile("bundle.js", string(serverBundle), false)
	if err != nil {
		return nil, fmt.Errorf("goja factory: compile: %w", err)
	}
	return &GojaVMFactory{prog: prog, polyfills: &polyfill.GojaPolyfillRegistry{}}, nil
}

// New implements framework.VMFactory.
func (f *GojaVMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false))

	vm := &VM{loop: loop, bridge: bridge}
	err := vm.run(func(rt *gojalib.Runtime) error {
		vm.rt = rt
		vm.jsi = rt.NewObject()
		rt.Set("__JSI__", vm.jsi)

		rt.Set("global", rt.GlobalObject())
		rt.Set("globalThis", rt.GlobalObject())
		rt.Set("console", map[string]any{
			"log":   func(c gojalib.FunctionCall) gojalib.Value { return logConsole(rt, "LOG", c) },
			"warn":  func(c gojalib.FunctionCall) gojalib.Value { return logConsole(rt, "WARN", c) },
			"error": func(c gojalib.FunctionCall) gojalib.Value { return logConsole(rt, "ERR", c) },
			"info":  func(c gojalib.FunctionCall) gojalib.Value { return logConsole(rt, "INFO", c) },
		})
		rt.Set("process", map[string]any{
			"env": map[string]any{"NODE_ENV": "production"},
		})
		rt.Set("performance", map[string]any{
			"now": func(c gojalib.FunctionCall) gojalib.Value { return rt.ToValue(0) },
		})

		for name, fn := range bridge.Globals {
			name, fn := name, fn
			rt.Set(name, func(c gojalib.FunctionCall) gojalib.Value {
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
