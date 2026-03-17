// Package sobek provides a Sobek-backed VM implementation for the GoJSX framework.
//
// Sobek (github.com/grafana/sobek) is a fork of Goja with the same API surface.
// This implementation uses a channel-based event loop to serialise all runtime
// access onto a single goroutine. Bridge async functions resolve via RunAsync so
// their Promise continuations are drained on the next checkpoint.
package sobek

import (
	"fmt"
	"sync"

	"gojsx/framework/contract"
	"gojsx/vm/eventloop"
	"gojsx/vm/sobek/polyfill"

	sobeklib "github.com/grafana/sobek"
)

// drainProg is a pre-compiled no-op that triggers Sobek's internal Promise
// microtask queue to drain. Called as a checkpoint after every loop task.
var drainProg, _ = sobeklib.Compile("drain.js", "0", false)

// VM wraps a Sobek runtime with a goroutine-serialised event loop.
type VM struct {
	rt     *sobeklib.Runtime
	loop   *eventloop.EventLoop
	jsi    *sobeklib.Object
	bridge contract.BridgeConfig
}

// VMPool manages a pool of pre-warmed VMs.
type VMPool struct {
	pool   sync.Pool
	bridge contract.BridgeConfig
	prog   *sobeklib.Program
}

// NewVMPool compiles the server bundle once and returns a pool.
// One VM is eagerly created to surface startup errors immediately.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	prog, err := sobeklib.Compile("bundle.js", serverBundle, false)
	if err != nil {
		return nil, fmt.Errorf("sobek: compile: %w", err)
	}
	p := &VMPool{bridge: bridge, prog: prog}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(prog, bridge)
			if err != nil {
				panic(fmt.Sprintf("sobek: pool create: %v", err))
			}
			return vm
		},
	}
	vm := p.pool.New()
	p.pool.Put(vm)
	return p, nil
}

// Acquire returns a VM from the pool.
func (p *VMPool) Acquire() *VM { return p.pool.Get().(*VM) }

// Release clears per-request state and returns the VM to the pool.
func (p *VMPool) Release(vm *VM) {
	_ = vm.ClearState()
	p.pool.Put(vm)
}

// Bridge returns the bridge config this pool was created with.
func (p *VMPool) Bridge() contract.BridgeConfig { return p.bridge }

// newVM creates a fresh event loop + runtime, installs globals/polyfills, runs the bundle.
func newVM(prog *sobeklib.Program, bridge contract.BridgeConfig) (*VM, error) {
	loop := eventloop.New(false)
	vm := &VM{loop: loop, bridge: bridge}

	var initErr error
	loop.RunSync(func() {
		rt := sobeklib.New()
		vm.rt = rt
		loop.SetCheckpoint(func() { rt.RunProgram(drainProg) }) //nolint:errcheck

		// Self-referential globals.
		rt.Set("global", rt.GlobalObject())     //nolint:errcheck
		rt.Set("globalThis", rt.GlobalObject()) //nolint:errcheck

		// Console.
		rt.Set("console", map[string]any{ //nolint:errcheck
			"log":   func(c sobeklib.FunctionCall) sobeklib.Value { return logConsole(rt, "LOG", c) },
			"warn":  func(c sobeklib.FunctionCall) sobeklib.Value { return logConsole(rt, "WARN", c) },
			"error": func(c sobeklib.FunctionCall) sobeklib.Value { return logConsole(rt, "ERR", c) },
			"info":  func(c sobeklib.FunctionCall) sobeklib.Value { return logConsole(rt, "INFO", c) },
		})

		// process + performance.
		rt.Set("process", map[string]any{ //nolint:errcheck
			"env": map[string]any{"NODE_ENV": "production"},
		})
		rt.Set("performance", map[string]any{ //nolint:errcheck
			"now": func(c sobeklib.FunctionCall) sobeklib.Value { return rt.ToValue(0) },
		})

		// __JSI__ bridge object.
		vm.jsi = rt.NewObject()
		rt.Set("__JSI__", vm.jsi) //nolint:errcheck

		// Global bridge functions (synchronous).
		for name, fn := range bridge.Globals {
			name, fn := name, fn
			rt.Set(name, func(c sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
				result, err := fn(exportArgs(c.Arguments))
				if err != nil {
					panic(rt.ToValue(err.Error()))
				}
				return rt.ToValue(result)
			})
		}

		// Polyfills.
		polyfill.Enable(rt)

		// Run the server bundle.
		if _, err := rt.RunProgram(prog); err != nil {
			initErr = fmt.Errorf("sobek: run bundle: %w", err)
		}
	})

	if initErr != nil {
		loop.Stop()
		return nil, initErr
	}
	return vm, nil
}

// run executes fn synchronously on the event loop.
func (vm *VM) run(fn func(rt *sobeklib.Runtime) error) error {
	var runErr error
	vm.loop.RunSync(func() {
		runErr = fn(vm.rt)
	})
	return runErr
}

func exportArgs(vals []sobeklib.Value) []interface{} {
	out := make([]interface{}, len(vals))
	for i, v := range vals {
		out[i] = v.Export()
	}
	return out
}

func logConsole(rt *sobeklib.Runtime, level string, c sobeklib.FunctionCall) sobeklib.Value {
	args := make([]string, len(c.Arguments))
	for i, a := range c.Arguments {
		args[i] = a.String()
	}
	fmt.Printf("[VM:%s] %v\n", level, args)
	return sobeklib.Undefined()
}
