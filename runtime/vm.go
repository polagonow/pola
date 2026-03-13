package runtime

import (
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

// GoFunc is the signature for Go functions exposed to the JS VM.
type GoFunc func(args []GoValue) (any, error)

// GoValue aliases goja.Value so callers don't need to import goja directly.
type GoValue = goja.Value

// BridgeConfig describes Go functions to expose inside the VM.
type BridgeConfig struct {
	// Globals are injected as bare global functions: fetchJSON("url")
	Globals map[string]GoFunc
	// Context functions are injected on a `ctx` object: ctx.getProducts()
	Context map[string]GoFunc
}

// VM wraps a single Goja runtime with the compiled server bundle.
type VM struct {
	rt *goja.Runtime
}

// Runtime exposes the underlying goja.Runtime for direct RunString calls.
func (vm *VM) Runtime() *goja.Runtime { return vm.rt }

// SetRequestContext injects per-request data as __request__ in the VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	vm.rt.Set("__request__", vm.rt.ToValue(ctx))
	return nil
}

// CallExport calls a named export from the server bundle (fallback path).
func (vm *VM) CallExport(name string, args ...goja.Value) (goja.Value, error) {
	fn, ok := goja.AssertFunction(vm.rt.Get(name))
	if !ok {
		return nil, fmt.Errorf("vm: %q is not a function", name)
	}
	return fn(goja.Undefined(), args...)
}

// RunString executes JS and returns the result.
// Each call to RunString causes Goja to drain its internal Promise job queue
// via leave() — this is the correct way to settle Promises in Goja.
func (vm *VM) RunString(src string) (goja.Value, error) {
	return vm.rt.RunString(src)
}

// VMPool manages a pool of pre-warmed VMs.
type VMPool struct {
	pool   sync.Pool
	bridge BridgeConfig
	prog   *goja.Program
}

// NewVMPool compiles the server bundle once and creates a pool of VMs.
func NewVMPool(serverBundle string, bridge BridgeConfig) (*VMPool, error) {
	prog, err := goja.Compile("bundle.js", serverBundle, false)
	if err != nil {
		return nil, fmt.Errorf("vm: compile: %w", err)
	}

	p := &VMPool{bridge: bridge, prog: prog}
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
func (p *VMPool) Acquire() *VM {
	return p.pool.Get().(*VM)
}

// Release returns a VM to the pool after resetting per-request state.
func (p *VMPool) Release(vm *VM) {
	// Clear per-request globals to avoid leaking between requests.
	vm.rt.Set("__request__", goja.Undefined())
	p.pool.Put(vm)
}

// newVM creates a fresh Goja runtime, injects the bridge, and runs the bundle.
func newVM(prog *goja.Program, bridge BridgeConfig) (*VM, error) {
	rt := goja.New()

	// Basic globals Goja doesn't provide out of the box.
	rt.Set("global", rt.GlobalObject())
	rt.Set("globalThis", rt.GlobalObject())

	// console
	rt.Set("console", map[string]any{
		"log":   func(c goja.FunctionCall) goja.Value { return logConsole(rt, "LOG", c) },
		"warn":  func(c goja.FunctionCall) goja.Value { return logConsole(rt, "WARN", c) },
		"error": func(c goja.FunctionCall) goja.Value { return logConsole(rt, "ERR", c) },
		"info":  func(c goja.FunctionCall) goja.Value { return logConsole(rt, "INFO", c) },
	})

	// process.env stub (react-dom checks NODE_ENV)
	rt.Set("process", map[string]any{
		"env": map[string]any{"NODE_ENV": "production"},
	})

	// setTimeout / clearTimeout stubs (react-dom uses these for error reporting)
	rt.Set("setTimeout", func(c goja.FunctionCall) goja.Value {
		// No-op: we don't have an event loop. Errors surface via Promise rejection.
		return rt.ToValue(0)
	})
	rt.Set("clearTimeout", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
	rt.Set("setInterval", func(c goja.FunctionCall) goja.Value { return rt.ToValue(0) })
	rt.Set("clearInterval", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })

	// performance.now() stub
	rt.Set("performance", map[string]any{
		"now": func(c goja.FunctionCall) goja.Value { return rt.ToValue(0) },
	})

	// ------------------------------------------------------------------ //
	// Go → JS bridge                                                       //
	// ------------------------------------------------------------------ //

	// Global functions
	for name, fn := range bridge.Globals {
		name, fn := name, fn // capture
		rt.Set(name, func(c goja.FunctionCall) goja.Value {
			result, err := fn(c.Arguments)
			if err != nil {
				panic(rt.ToValue(err.Error()))
			}
			return rt.ToValue(result)
		})
	}

	// ctx object
	ctxObj := rt.NewObject()
	for name, fn := range bridge.Context {
		name, fn := name, fn
		ctxObj.Set(name, func(c goja.FunctionCall) goja.Value {
			result, err := fn(c.Arguments)
			if err != nil {
				panic(rt.ToValue(err.Error()))
			}
			return rt.ToValue(result)
		})
	}
	rt.Set("ctx", ctxObj)

	// Run the compiled bundle (polyfills + server-dom IIFE + pages CJS)
	if _, err := rt.RunProgram(prog); err != nil {
		return nil, fmt.Errorf("vm: run program: %w", err)
	}

	return &VM{rt: rt}, nil
}

func logConsole(rt *goja.Runtime, level string, c goja.FunctionCall) goja.Value {
	args := make([]string, len(c.Arguments))
	for i, a := range c.Arguments {
		args[i] = a.String()
	}
	fmt.Printf("[VM:%s] %v\n", level, args)
	return goja.Undefined()
}
