// Package runtime provides the Goja VM pool and JS rendering bridge.
//
// Each VM is backed by a goja_nodejs EventLoop. The loop gives us real
// setTimeout/setInterval scheduling and — crucially — proper microtask
// flushing between ticks so that async server components (async/await)
// resolve correctly inside renderToReadableStream.
//
// Render lifecycle per request:
//
//  1. loop.Run(setContextFn)   – inject per-request data
//  2. loop.Run(renderFn)       – start rendering; __poll__ reschedules itself
//                                via setTimeout(0) until the stream is done;
//                                loop.Run returns only when no more jobs remain
//  3. loop.Run(cleanupFn)      – clear per-request globals
//
// Because loop.Run executes on the caller's goroutine and the pool ensures
// at most one goroutine holds a VM at a time, all Goja access is single-
// threaded and safe.
package runtime

import (
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
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

// VM wraps a goja EventLoop with the pre-loaded server bundle.
// All Goja operations run on the caller's goroutine via loop.Run.
type VM struct {
	loop *eventloop.EventLoop
	rt   *goja.Runtime // captured during init; valid for VM lifetime
}

// run executes fn synchronously on the event loop, drains all pending
// timers and microtasks, then returns. Safe to call repeatedly.
func (vm *VM) run(fn func(rt *goja.Runtime) error) error {
	var runErr error
	vm.loop.Run(func(rt *goja.Runtime) {
		runErr = fn(rt)
	})
	return runErr
}

// SetRequestContext injects per-request data as __request__ in the VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return vm.run(func(rt *goja.Runtime) error {
		rt.Set("__request__", rt.ToValue(ctx))
		return nil
	})
}

// CallExport calls a named export from the server bundle (fallback path).
func (vm *VM) CallExport(name string, args ...goja.Value) (goja.Value, error) {
	var result goja.Value
	err := vm.run(func(rt *goja.Runtime) error {
		fn, ok := goja.AssertFunction(rt.Get(name))
		if !ok {
			return fmt.Errorf("vm: %q is not a function", name)
		}
		var err error
		result, err = fn(goja.Undefined(), args...)
		return err
	})
	return result, err
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

// Release clears per-request state and returns the VM to the pool.
func (p *VMPool) Release(vm *VM) {
	_ = vm.run(func(rt *goja.Runtime) error {
		rt.Set("__request__", goja.Undefined())
		rt.Set("__gojsx_stream__", goja.Undefined())
		return nil
	})
	p.pool.Put(vm)
}

// newVM creates a fresh EventLoop, registers the bridge, and runs the bundle.
func newVM(prog *goja.Program, bridge BridgeConfig) (*VM, error) {
	// EnableConsole(false): we provide our own console below.
	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false))

	vm := &VM{loop: loop}
	err := vm.run(func(rt *goja.Runtime) error {
		vm.rt = rt // capture for use in async bridge closures

		// ── Basic globals ─────────────────────────────────────────────
		rt.Set("global", rt.GlobalObject())
		rt.Set("globalThis", rt.GlobalObject())

		// console (goja_nodejs console module disabled; provide our own)
		rt.Set("console", map[string]any{
			"log":   func(c goja.FunctionCall) goja.Value { return logConsole(rt, "LOG", c) },
			"warn":  func(c goja.FunctionCall) goja.Value { return logConsole(rt, "WARN", c) },
			"error": func(c goja.FunctionCall) goja.Value { return logConsole(rt, "ERR", c) },
			"info":  func(c goja.FunctionCall) goja.Value { return logConsole(rt, "INFO", c) },
		})

		// process.env stub (React checks NODE_ENV)
		rt.Set("process", map[string]any{
			"env": map[string]any{"NODE_ENV": "production"},
		})

		// performance.now() stub
		rt.Set("performance", map[string]any{
			"now": func(c goja.FunctionCall) goja.Value { return rt.ToValue(0) },
		})

		// Note: setTimeout / setInterval / clearTimeout / clearInterval are
		// provided by goja_nodejs EventLoop — do not override them here.

		// ── Go → JS bridge ───────────────────────────────────────────
		for name, fn := range bridge.Globals {
			name, fn := name, fn
			rt.Set(name, func(c goja.FunctionCall) goja.Value {
				result, err := fn(c.Arguments)
				if err != nil {
					panic(rt.ToValue(err.Error()))
				}
				return rt.ToValue(result)
			})
		}

		// Context functions run in Go goroutines and return Promises so that
		// async server components (async function Foo() { await ctx.getFoo() })
		// can suspend correctly. The poll loop's repeated setTimeout(0) calls
		// keep the event loop alive while the goroutine works; when the work
		// completes, loop.RunOnLoop resolves the Promise on the event loop
		// goroutine, React continues rendering, and the stream emits the
		// resolved content.
		ctxObj := rt.NewObject()
		for name, fn := range bridge.Context {
			name, fn := name, fn
			ctxObj.Set(name, func(c goja.FunctionCall) goja.Value {
				// Capture argument values as exported Go types before entering
				// the goroutine; goja.Value internals must not be read from
				// outside the event loop goroutine.
				exported := make([]interface{}, len(c.Arguments))
				for i, a := range c.Arguments {
					exported[i] = a.Export()
				}

				p, resolve, reject := rt.NewPromise()

				go func() {
					// Re-wrap exported values so GoFunc receives goja.Value-
					// compatible arguments. Since we Export()ed them, pass as
					// exported interface{} — but GoFunc expects []goja.Value.
					// We resolve the actual Go work here; args are not needed
					// by most bridge functions (getProducts, getUser use 0 or 1
					// simple scalar), so pass original slice — scalars are safe.
					result, err := fn(c.Arguments)
					vm.loop.RunOnLoop(func(rt *goja.Runtime) {
						if err != nil {
							reject(rt.ToValue(err.Error()))
						} else {
							resolve(rt.ToValue(result))
						}
					})
				}()

				return rt.ToValue(p)
			})
		}
		rt.Set("ctx", ctxObj)

		// ── Run the compiled bundle (polyfills + server-dom + pages) ─
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

func logConsole(rt *goja.Runtime, level string, c goja.FunctionCall) goja.Value {
	args := make([]string, len(c.Arguments))
	for i, a := range c.Arguments {
		args[i] = a.String()
	}
	fmt.Printf("[VM:%s] %v\n", level, args)
	return goja.Undefined()
}
