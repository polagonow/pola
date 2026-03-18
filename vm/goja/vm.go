// Package goja provides a Goja-backed VM implementation for the Pola framework.
//
// Each VM is backed by a goja_nodejs EventLoop. The loop gives us real
// setTimeout/setInterval scheduling and — crucially — proper microtask
// flushing between ticks so that async server components (async/await)
// resolve correctly inside renderToReadableStream.
package goja

import (
	"fmt"
	"sync"

	"github.com/polagonow/pola/framework/contract"
	"github.com/polagonow/pola/framework/globals"
	polyfill "github.com/polagonow/pola/vm/goja/polyfill"

	gojalib "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// VM wraps a goja EventLoop with the pre-loaded server bundle.
// All Goja operations run on the caller's goroutine via loop.Run.
type VM struct {
	loop   *eventloop.EventLoop
	rt     *gojalib.Runtime // captured during init; valid for VM lifetime
	jsi    *gojalib.Object  // the persistent __JSI__ object; mutated per-render
	bridge contract.BridgeConfig
}

// run executes fn synchronously on the event loop, drains all pending
// timers and microtasks, then returns.
func (vm *VM) run(fn func(rt *gojalib.Runtime) error) error {
	var runErr error
	vm.loop.Run(func(rt *gojalib.Runtime) {
		runErr = fn(rt)
	})
	return runErr
}

// SetJSI swaps the functions on the persistent __JSI__ object for this render.
func (vm *VM) SetJSI(funcs map[string]contract.GoFunc) error {
	return vm.run(func(rt *gojalib.Runtime) error {
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		for name, fn := range funcs {
			vm.jsi.Set(name, func(c gojalib.FunctionCall) gojalib.Value { //nolint:errcheck
				args := exportArgs(c.Arguments)
				p, resolve, reject := rt.NewPromise()
				go func() {
					result, err := fn(args)
					vm.loop.RunOnLoop(func(rt *gojalib.Runtime) {
						if err != nil {
							_ = reject(rt.ToValue(err.Error()))
						} else {
							_ = resolve(rt.ToValue(result))
						}
					})
				}()
				return rt.ToValue(p)
			})
		}
		return nil
	})
}

// SetRequestContext injects per-request data as __REQUEST__ in the VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return vm.run(func(rt *gojalib.Runtime) error {
		rt.Set(globals.RequestContext, rt.ToValue(ctx)) //nolint:errcheck
		return nil
	})
}

// CallExport calls a named export from the server bundle.
func (vm *VM) CallExport(name string, args ...gojalib.Value) (gojalib.Value, error) {
	var result gojalib.Value
	err := vm.run(func(rt *gojalib.Runtime) error {
		fn, ok := gojalib.AssertFunction(rt.Get(name))
		if !ok {
			return fmt.Errorf("vm: %q is not a function", name)
		}
		var err error
		result, err = fn(gojalib.Undefined(), args...)
		return err
	})
	return result, err
}

// VMPool manages a pool of pre-warmed VMs.
type vmPool struct {
	pool   sync.Pool
	bridge contract.BridgeConfig
	prog   *gojalib.Program
}

// NewVMPool compiles the server bundle once and creates a pool of VMs.
func newVMPool(serverBundle string, bridge contract.BridgeConfig) (*vmPool, error) {
	prog, err := gojalib.Compile("bundle.js", serverBundle, false)
	if err != nil {
		return nil, fmt.Errorf("vm: compile: %w", err)
	}

	p := &vmPool{bridge: bridge, prog: prog}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(prog, bridge)
			if err != nil {
				panic(fmt.Sprintf("vm: pool create: %v", err))
			}
			return vm
		},
	}

	// Eagerly create one VM to catch startup errors immediately.
	vm := p.pool.New()
	p.pool.Put(vm)
	return p, nil
}

// Acquire returns a VM from the pool.
func (p *vmPool) Acquire() *VM {
	return p.pool.Get().(*VM)
}

// Release clears per-request state and returns the VM to the pool.
func (p *vmPool) Release(vm *VM) {
	_ = vm.run(func(rt *gojalib.Runtime) error {
		rt.Set(globals.RequestContext, gojalib.Undefined()) //nolint:errcheck
		rt.Set(globals.StreamHandle, gojalib.Undefined())   //nolint:errcheck
		for _, key := range vm.jsi.Keys() {
			vm.jsi.Delete(key) //nolint:errcheck
		}
		return nil
	})
	p.pool.Put(vm)
}

// Bridge returns the bridge config this pool was created with.
func (p *vmPool) Bridge() contract.BridgeConfig {
	return p.bridge
}

// newVM creates a fresh EventLoop, registers the bridge, and runs the bundle.
func newVM(prog *gojalib.Program, bridge contract.BridgeConfig) (*VM, error) {
	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false))

	vm := &VM{loop: loop, bridge: bridge}
	err := vm.run(func(rt *gojalib.Runtime) error {
		vm.rt = rt
		vm.jsi = rt.NewObject()
		rt.Set(globals.BridgeObject, vm.jsi)    //nolint:errcheck
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

		if err := polyfill.Enable(rt); err != nil {
			return fmt.Errorf("vm: polyfill: %w", err)
		}

		if _, err := rt.RunProgram(prog); err != nil {
			return fmt.Errorf("vm: run program: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func exportArgs(vals []gojalib.Value) []interface{} {
	out := make([]interface{}, len(vals))
	for i, v := range vals {
		out[i] = v.Export()
	}
	return out
}

func logConsole(level string, c gojalib.FunctionCall) gojalib.Value {
	args := make([]string, len(c.Arguments))
	for i, a := range c.Arguments {
		args[i] = a.String()
	}
	fmt.Printf("[VM:%s] %v\n", level, args)
	return gojalib.Undefined()
}
