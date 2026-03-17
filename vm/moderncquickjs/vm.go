// Package moderncquickjs provides a modernc.org/quickjs-backed VM implementation
// for the GoJSX framework. This is a pure-Go (no CGo) QuickJS binding.
//
// Bridge functions (both Globals and Context) are invoked synchronously inside
// the JS event loop. The request goroutine blocks until each bridge call returns.
//
// # Async support
//
// modernc.org/quickjs does not expose JS_ExecutePendingJob through its public
// API. drainNativeJobs accesses the VM's unexported fields via unsafe pointer
// arithmetic to call XJS_ExecutePendingJob directly. This drains the native
// QuickJS job queue (async/await continuations, native Promise callbacks) that
// __drainMicrotasks__ cannot reach. It is called by the event loop checkpoint
// after every task so that async React server components complete.
//
// VM struct layout (64-bit):
//
//	{ cContext uintptr @0; goFuncs @8; int32_16 @16; int32_2 @32; runtime *runtime @48; ... }
//	runtime { cRuntime uintptr @0; tls *libc.TLS @8 }
package moderncquickjs

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	lib "modernc.org/libquickjs"
	libc "modernc.org/libc"
	mquickjs "modernc.org/quickjs"

	"gojsx/framework"
	"gojsx/framework/contract"
	"gojsx/vm/moderncquickjs/polyfill"
)

// renderHelpersJS defines JS helpers used by the Go-driven render loop.
// The outer iteration happens in render.go (DrainStream) so the event-loop
// checkpoint runs __drainMicrotasks__ + drainNativeJobs between each pull.
const renderHelpersJS = `
globalThis.__startRender__ = function(exportName, propsJSON) {
	globalThis.__rscStream__ = __render__(exportName, propsJSON);
	globalThis.__rscDec__    = new TextDecoder();
	void 0;
};

globalThis.__pullOnce__ = function() {
	var s   = globalThis.__rscStream__;
	var dec = globalThis.__rscDec__;
	s._start();
	__drainMicrotasks__();
	s._pull();
	__drainMicrotasks__();
	var chunks = s._controller._chunks.splice(0);
	var closed = s._controller._closed;
	for (var i = 0; i < chunks.length; i++) {
		__outputChunk__(dec.decode(chunks[i]));
	}
	return closed && chunks.length === 0;
};

globalThis.__clearRenderStream__ = function() {
	globalThis.__rscStream__ = undefined;
	globalThis.__rscDec__    = undefined;
	void 0;
};
`

// jsiWrapperJS is the JS template for a single __JSI__ bridge wrapper.
// %s is substituted with the JSON-encoded function name (a quoted string).
const jsiWrapperJS = `__JSI__[%s] = function() {
	var args = Array.prototype.slice.call(arguments);
	var json = __mqjs_call__(%s, JSON.stringify(args));
	var result = JSON.parse(json);
	if (result && typeof result === 'object' && result.__mqjs_error__) {
		throw new Error(result.__mqjs_error__);
	}
	return result;
};`

// VM holds a modernc.org/quickjs VM. All JS operations are serialised via the event loop.
type VM struct {
	inner         *mquickjs.VM
	loop          *eventLoop
	globalFuncs   map[string]contract.GoFunc
	perReqFuncs   map[string]contract.GoFunc
	currentWriter framework.StreamWriter
	wroteAny      bool
}

func (vm *VM) run(fn func()) {
	vm.loop.RunSync(fn)
}

// vmRuntimeLayout mirrors the unexported modernc.org/quickjs.runtime struct
// for unsafe field access in drainNativeJobs.
type vmRuntimeLayout struct {
	cRuntime uintptr
	tls      *libc.TLS
}

// drainNativeJobs runs all pending QuickJS jobs (native async/await
// continuations). Accesses VM internals via unsafe; must be called on the
// event loop goroutine.
func drainNativeJobs(vm *mquickjs.VM) {
	vmPtr := unsafe.Pointer(vm)
	runtimePtr := *(*uintptr)(unsafe.Add(vmPtr, 48)) // runtime field at offset 48
	rt := (*vmRuntimeLayout)(unsafe.Pointer(runtimePtr))
	for lib.XJS_ExecutePendingJob(rt.tls, rt.cRuntime, 0) > 0 {
	}
}

// VMPool manages a pool of pre-warmed VMs.
type VMPool struct {
	pool   sync.Pool
	bridge contract.BridgeConfig
	src    string
}

// NewVMPool creates a pool backed by server bundle + bridge config.
// One VM is eagerly created to surface startup errors immediately.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	p := &VMPool{bridge: bridge, src: serverBundle}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(serverBundle, bridge)
			if err != nil {
				panic(fmt.Sprintf("moderncquickjs: pool create: %v", err))
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

// evalOrErr evaluates src and returns any JS exception as a Go error.
func evalOrErr(inner *mquickjs.VM, src string) error {
	_, err := inner.Eval(src, mquickjs.EvalGlobal)
	return err
}

// newVM creates a runtime, installs globals/polyfills/bundle.
// All operations including VM creation run on the event loop goroutine so that
// modernc.org/quickjs sees a consistent goroutine for every JS call.
func newVM(src string, bridge contract.BridgeConfig) (*VM, error) {
	loop := newEventLoop()

	var vm *VM
	var initErr error
	loop.RunSync(func() {
		inner, err := mquickjs.NewVM()
		if err != nil {
			initErr = fmt.Errorf("moderncquickjs: new vm: %w", err)
			return
		}
		vm = &VM{
			inner:       inner,
			loop:        loop,
			globalFuncs: bridge.Globals,
		}

		// ── Base globals ──────────────────────────────────────────────
		// Trailing "void 0" avoids returning a circular globalThis value to Go.
		if err := evalOrErr(inner, `
globalThis.global = globalThis;
globalThis.globalThis = globalThis;
void 0;
`); err != nil {
			initErr = fmt.Errorf("moderncquickjs: globals: %w", err)
			return
		}

		// ── Console ───────────────────────────────────────────────────
		if err := inner.RegisterFunc("__mqjs_log__", func(level, msg string) {
			fmt.Printf("[VM:%s] %s\n", level, msg)
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register __mqjs_log__: %w", err)
			return
		}
		if err := evalOrErr(inner, `
globalThis.console = {
	log:   function() { __mqjs_log__("LOG",  Array.prototype.slice.call(arguments).join(' ')); },
	warn:  function() { __mqjs_log__("WARN", Array.prototype.slice.call(arguments).join(' ')); },
	error: function() { __mqjs_log__("ERR",  Array.prototype.slice.call(arguments).join(' ')); },
	info:  function() { __mqjs_log__("INFO", Array.prototype.slice.call(arguments).join(' ')); },
};
void 0;`); err != nil {
			initErr = fmt.Errorf("moderncquickjs: console: %w", err)
			return
		}

		// ── Process / performance stubs ───────────────────────────────
		if err := evalOrErr(inner, `
globalThis.process = { env: { NODE_ENV: "production" } };
globalThis.performance = { now: function() { return 0; } };
void 0;
`); err != nil {
			initErr = fmt.Errorf("moderncquickjs: process: %w", err)
			return
		}

		// ── Bridge dispatcher: __mqjs_call__(name, argsJSON) → resultJSON ──
		//
		// Dispatches both global and per-request bridge functions.
		// Runs on the event loop goroutine during Eval, so no additional locking needed.
		if err := inner.RegisterFunc("__mqjs_call__", func(name, argsJSON string) string {
			var fn contract.GoFunc
			if f, ok := vm.globalFuncs[name]; ok {
				fn = f
			} else if f, ok := vm.perReqFuncs[name]; ok {
				fn = f
			} else {
				b, _ := json.Marshal(map[string]any{"__mqjs_error__": "bridge: " + name + " not found"})
				return string(b)
			}
			var args []interface{}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				b, _ := json.Marshal(map[string]any{"__mqjs_error__": "bridge args: " + err.Error()})
				return string(b)
			}
			result, err := fn(args)
			if err != nil {
				b, _ := json.Marshal(map[string]any{"__mqjs_error__": err.Error()})
				return string(b)
			}
			b, _ := json.Marshal(result)
			return string(b)
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register __mqjs_call__: %w", err)
			return
		}

		// ── __JSI__: plain object for per-request bridge functions ────
		// Properties are added by SetBridgeFunctions and cleared by ClearState,
		// matching the quickjsgo pattern.
		if err := evalOrErr(inner, "globalThis.__JSI__ = {}; void 0;"); err != nil {
			initErr = fmt.Errorf("moderncquickjs: jsi: %w", err)
			return
		}

		// ── Global bridge functions as JS wrappers ────────────────────
		// Each wrapper calls __mqjs_call__ and parses the JSON result.
		for name := range bridge.Globals {
			nameJSON, _ := json.Marshal(name)
			script := fmt.Sprintf(`globalThis[%s] = function() {
	var args = Array.prototype.slice.call(arguments);
	var json = __mqjs_call__(%s, JSON.stringify(args));
	var result = JSON.parse(json);
	if (result && typeof result === 'object' && result.__mqjs_error__) {
		throw new Error(result.__mqjs_error__);
	}
	return result;
};
void 0;`, string(nameJSON), string(nameJSON))
			if err := evalOrErr(inner, script); err != nil {
				initErr = fmt.Errorf("moderncquickjs: install global bridge %q: %w", name, err)
				return
			}
		}

		// ── __outputChunk__ ───────────────────────────────────────────
		// Called synchronously by __pullOnce__ for each rendered chunk.
		if err := inner.RegisterFunc("__outputChunk__", func(text string) {
			if vm.currentWriter != nil {
				vm.currentWriter.WriteRaw([]byte(text)) //nolint:errcheck
				vm.currentWriter.Flush()
				vm.wroteAny = true
			}
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register __outputChunk__: %w", err)
			return
		}

		// ── Polyfills ─────────────────────────────────────────────────
		if err := polyfill.Enable(inner); err != nil {
			initErr = fmt.Errorf("moderncquickjs: polyfills: %w", err)
			return
		}

		// ── Server bundle ─────────────────────────────────────────────
		if err := evalOrErr(inner, src); err != nil {
			initErr = fmt.Errorf("moderncquickjs: run bundle: %w", err)
			return
		}

		// ── Render helpers ────────────────────────────────────────────
		if err := evalOrErr(inner, renderHelpersJS); err != nil {
			initErr = fmt.Errorf("moderncquickjs: render helpers: %w", err)
			return
		}

		// ── Wire checkpoint ───────────────────────────────────────────
		// After every event-loop task: drain native JS jobs (async/await)
		// then drain our custom microtask queue.
		loop.setCheckpoint(func() {
			drainNativeJobs(vm.inner)
			evalOrErr(vm.inner, "__drainMicrotasks__();") //nolint:errcheck
		})
	})

	if initErr != nil {
		if vm != nil {
			loop.RunSync(func() { vm.inner.Close() })
		}
		loop.Stop()
		return nil, initErr
	}
	return vm, nil
}
