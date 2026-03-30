//go:build moderncquickjs

// Package moderncquickjs provides a modernc.org/quickjs-backed JSEngine for the
// Pola framework. Pure-Go QuickJS (no CGo) via modernc.org/quickjs.
//
// Bridge calls are dispatched synchronously through __mqjs_call__. The render
// loop is driven from Go: __startRender__ stores the RSC ReadableStream, then
// __pullOnce__ is called repeatedly — each call is a separate event-loop task
// so the checkpoint (drainNativeJobs + __drainMicrotasks__) fires between
// iterations, allowing async server components to advance.
//
// # Async support
//
// modernc.org/quickjs does not expose JS_ExecutePendingJob through its public
// API. drainNativeJobs accesses the VM's unexported fields via unsafe pointer
// arithmetic to call XJS_ExecutePendingJob directly.
//
// Build tag: moderncquickjs
package moderncquickjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
	"unsafe"

	libc "modernc.org/libc"
	lib "modernc.org/libquickjs"
	mquickjs "modernc.org/quickjs"

	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/eventloop"
	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/vmpool"
)

// ── JS templates ──────────────────────────────────────────────────────────────

var (
	renderHelpersJSTmpl = template.Must(template.New("renderHelpers").Parse(`
globalThis.{{.StartRenderFn}} = function(exportName, propsJSON) {
	globalThis.{{.RSCStreamVar}} = {{.RenderFn}}(exportName, propsJSON);
	globalThis.{{.RSCDecoderVar}} = new TextDecoder();
	void 0;
};

globalThis.{{.PullOnceFn}} = function() {
	var s   = globalThis.{{.RSCStreamVar}};
	var dec = globalThis.{{.RSCDecoderVar}};
	s._start();
	{{.DrainMicrotasksFn}}();
	s._pull();
	{{.DrainMicrotasksFn}}();
	var chunks = s._controller._chunks.splice(0);
	var closed = s._controller._closed;
	for (var i = 0; i < chunks.length; i++) {
		{{.OutputChunk}}(dec.decode(chunks[i]));
	}
	return closed && chunks.length === 0;
};

globalThis.{{.ClearRenderStreamFn}} = function() {
	globalThis.{{.RSCStreamVar}} = undefined;
	globalThis.{{.RSCDecoderVar}} = undefined;
	void 0;
};
`))

	jsiWrapperJSTmpl = template.Must(template.New("jsiWrapper").Parse(`{{.BridgeObject}}[{{.NameJSON}}] = function() {
	var args = Array.prototype.slice.call(arguments);
	var json = {{.MQJSCallFn}}({{.NameJSON}}, JSON.stringify(args));
	var result = JSON.parse(json);
	if (result && typeof result === 'object' && result.{{.MQJSErrorKey}}) {
		throw new Error(result.{{.MQJSErrorKey}});
	}
	return result;
};`))

	renderHelpersJS string
)

func init() {
	var b strings.Builder
	if err := renderHelpersJSTmpl.Execute(&b, struct {
		StartRenderFn, RSCStreamVar, RenderFn, RSCDecoderVar            string
		PullOnceFn, DrainMicrotasksFn, OutputChunk, ClearRenderStreamFn string
	}{
		globals.StartRenderFn, globals.RSCStreamVar, globals.RenderFn, globals.RSCDecoderVar,
		globals.PullOnceFn, globals.DrainMicrotasksFn, globals.OutputChunk, globals.ClearRenderStreamFn,
	}); err != nil {
		panic("moderncquickjs: renderHelpers template: " + err.Error())
	}
	renderHelpersJS = b.String()
}

// ── unsafe intrinsics ─────────────────────────────────────────────────────────

// vmRuntimeLayout mirrors the unexported modernc.org/quickjs.runtime struct
// for unsafe field access in drainNativeJobs.
//
// VM struct layout (64-bit):
//
//	{ cContext uintptr @0; goFuncs @8; int32_16 @16; int32_2 @32; runtime *runtime @48; ... }
//	runtime { cRuntime uintptr @0; tls *libc.TLS @8 }
type vmRuntimeLayout struct {
	cRuntime uintptr
	tls      *libc.TLS
}

// drainNativeJobs runs all pending QuickJS jobs (native async/await
// continuations). Accesses VM internals via unsafe; must be called on the
// event loop goroutine.
func drainNativeJobs(vm *mquickjs.VM) {
	vmPtr := unsafe.Pointer(vm)
	runtimePtr := *(*uintptr)(unsafe.Add(vmPtr, 48))     // runtime field at offset 48
	rt := (*vmRuntimeLayout)(unsafe.Pointer(runtimePtr)) //nolint:govet
	for lib.XJS_ExecutePendingJob(rt.tls, rt.cRuntime, 0) > 0 {
	}
}

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine is a moderncquickjs-backed JSEngine. It implements core.JSEngine and
// core.SSRPoolFactory.
type Engine struct {
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// NewEngine returns a stateless Engine (no bundle pre-compilation needed).
func NewEngine() *Engine { return &Engine{} }

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "moderncquickjs" }

// RequiredPolyfills implements core.JSEngine.
// moderncquickjs needs all polyfills including Promise (no native Promise).
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
	return []core.PolyfillID{
		polyfill.NodeGlobals,
		polyfill.ConsoleBridge,
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
		polyfill.Promise,
	}
}

// NewRuntime implements core.JSEngine. Returns an error — use NewSSRPool for
// bundle-backed runtimes; the SSR pipeline calls that instead.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	return nil, fmt.Errorf("moderncquickjs: use NewSSRPool to create bundle-backed runtimes")
}

// NewSSRPool implements core.SSRPoolFactory. Compiles the server bundle into a
// pool of moderncquickjs Runtimes.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
	pool, err := newVMPool(string(bundle), e.logger)
	if err != nil {
		return nil, err
	}
	return &mqjsSSRPool{pool: pool}, nil
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a modernc.org/quickjs VM with an event loop so all JS
// operations serialise onto a single goroutine.
type Runtime struct {
	inner         *mquickjs.VM
	loop          *eventloop.EventLoop
	diFuncs       map[string]func(args []any) (any, error)
	currentWriter core.StreamWriter
	wroteAny      bool
}

func (r *Runtime) run(fn func()) {
	r.loop.RunSync(fn)
}

func evalOrErr(inner *mquickjs.VM, src string) error {
	_, err := inner.Eval(src, mquickjs.EvalGlobal)
	return err
}

func newRuntime(src string, logger core.Logger) (*Runtime, error) {
	loop := eventloop.New(false)

	var r *Runtime
	var initErr error
	loop.RunSync(func() {
		inner, err := mquickjs.NewVM()
		if err != nil {
			initErr = fmt.Errorf("moderncquickjs: new vm: %w", err)
			return
		}
		r = &Runtime{inner: inner, loop: loop}

		// ── Console bridge ────────────────────────────────────────────
		// Install __pola_log__ before ConsoleBridge polyfill.
		if err := inner.RegisterFunc(globals.PolaLogFn, func(level, msg string) {
			polyfill.LogAtLevel(logger, "moderncquickjs", level, msg)
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register %s: %w", globals.PolaLogFn, err)
			return
		}

		// ── Bridge dispatcher ─────────────────────────────────────────
		// __mqjs_call__(name, argsJSON) → resultJSON
		// Synchronous; called by __DEPENDENCY_INJECTION__[name] JS wrappers.
		if err := inner.RegisterFunc(globals.MQJSCallFn, func(name, argsJSON string) string {
			var fn func(args []any) (any, error)
			if r.diFuncs != nil {
				fn = r.diFuncs[name]
			}
			if fn == nil {
				b, _ := json.Marshal(map[string]any{globals.MQJSErrorKey: "bridge: " + name + " not found"})
				return string(b)
			}
			var args []any
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				b, _ := json.Marshal(map[string]any{globals.MQJSErrorKey: "bridge args: " + err.Error()})
				return string(b)
			}
			result, err := fn(args)
			if err != nil {
				b, _ := json.Marshal(map[string]any{globals.MQJSErrorKey: err.Error()})
				return string(b)
			}
			b, _ := json.Marshal(result)
			return string(b)
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register %s: %w", globals.MQJSCallFn, err)
			return
		}

		// ── __DEPENDENCY_INJECTION__ placeholder ──────────────────────
		if err := evalOrErr(inner, "globalThis."+globals.BridgeObject+" = {}; void 0;"); err != nil {
			initErr = fmt.Errorf("moderncquickjs: bridge object: %w", err)
			return
		}

		// ── __outputChunk__ ───────────────────────────────────────────
		if err := inner.RegisterFunc(globals.OutputChunk, func(text string) {
			if r.currentWriter != nil {
				r.currentWriter.WriteRaw([]byte(text)) //nolint:errcheck
				r.currentWriter.Flush()
				r.wroteAny = true
			}
		}, false); err != nil {
			initErr = fmt.Errorf("moderncquickjs: register %s: %w", globals.OutputChunk, err)
			return
		}

		// ── Polyfills (NodeGlobals + ConsoleBridge before bundle) ─────
		reg := polyfill.DefaultRegistry()
		for _, pf := range reg.Get(
			polyfill.NodeGlobals,
			polyfill.ConsoleBridge,
			polyfill.MicrotaskQueue,
			polyfill.TextEncoding,
			polyfill.MessageChannel,
			polyfill.ReadableStream,
			polyfill.AbortController,
			polyfill.WebpackRequire,
			polyfill.Promise,
		) {
			if err := evalOrErr(inner, pf.Source); err != nil {
				initErr = fmt.Errorf("moderncquickjs: polyfill %s: %w", pf.ID, err)
				return
			}
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

		// ── Checkpoint ────────────────────────────────────────────────
		// After every event-loop task: drain native JS jobs then microtask queue.
		loop.SetCheckpoint(func() {
			drainNativeJobs(r.inner)
			evalOrErr(r.inner, globals.DrainMicrotasksFn+"();") //nolint:errcheck
		})
	})

	if initErr != nil {
		if r != nil {
			loop.RunSync(func() { _ = r.inner.Close() })
		}
		loop.Stop()
		return nil, initErr
	}
	return r, nil
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	var runErr error
	r.run(func() {
		v, err := r.inner.Eval(script, mquickjs.EvalGlobal)
		if err != nil {
			runErr = err
			return
		}
		result = v
	})
	return result, runErr
}

// Call implements core.JSRuntime.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("moderncquickjs: marshal args: %w", err)
	}
	script := fmt.Sprintf("(function(){ var __args = %s; return %s.apply(null, __args); })()", string(argsJSON), fn)
	return r.Eval(script)
}

// Set implements core.JSRuntime.
func (r *Runtime) Set(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("moderncquickjs: marshal value for %s: %w", name, err)
	}
	var runErr error
	r.run(func() {
		runErr = evalOrErr(r.inner, "globalThis."+name+" = ("+string(data)+");")
	})
	return runErr
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.run(func() { _ = r.inner.Close() })
	r.loop.Stop()
}

// SetRequestContext implements core.SSRRuntime.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("moderncquickjs: marshal request context: %w", err)
	}
	var runErr error
	r.run(func() {
		_, runErr = r.inner.Eval(
			"globalThis."+globals.RequestContext+" = ("+string(b)+");",
			mquickjs.EvalGlobal,
		)
	})
	return runErr
}

// SetDependencyInjection installs DI functions on __DEPENDENCY_INJECTION__.
// Each function is dispatched synchronously through the __mqjs_call__ bridge.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	var runErr error
	r.run(func() {
		r.diFuncs = funcs

		// Clear existing __DEPENDENCY_INJECTION__ properties.
		if _, err := r.inner.Eval(
			"Object.keys("+globals.BridgeObject+").forEach(function(k) { delete "+globals.BridgeObject+"[k]; });",
			mquickjs.EvalGlobal,
		); err != nil {
			runErr = fmt.Errorf("moderncquickjs: clear DI: %w", err)
			return
		}

		// Install a synchronous JS wrapper for each function.
		for name := range funcs {
			nameJSON, _ := json.Marshal(name)
			var scriptBuf strings.Builder
			if err := jsiWrapperJSTmpl.Execute(&scriptBuf, struct {
				BridgeObject, NameJSON, MQJSCallFn, MQJSErrorKey string
			}{globals.BridgeObject, string(nameJSON), globals.MQJSCallFn, globals.MQJSErrorKey}); err != nil {
				runErr = fmt.Errorf("moderncquickjs: wrapper template: %w", err)
				return
			}
			if _, err := r.inner.Eval(scriptBuf.String(), mquickjs.EvalGlobal); err != nil {
				runErr = fmt.Errorf("moderncquickjs: install DI bridge %q: %w", name, err)
				return
			}
		}
	})
	return runErr
}

// CallRenderFunction implements core.SSRRuntime. Stores the render parameters;
// the actual JS execution happens in DrainStream.
func (r *Runtime) CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error) {
	return &mqjsStreamHandle{exportName: exportName, propsJSON: propsJSON}, nil
}

// DrainStream implements core.SSRRuntime.
func (r *Runtime) DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error) {
	h, ok := handle.(*mqjsStreamHandle)
	if !ok {
		return false, fmt.Errorf("moderncquickjs: DrainStream: unexpected handle type %T", handle)
	}
	return r.drainStream(h.exportName, h.propsJSON, w)
}

func (r *Runtime) drainStream(exportName, propsJSON string, w core.StreamWriter) (bool, error) {
	var err error

	// Start render: install stream + TextDecoder in JS globals.
	r.run(func() {
		r.currentWriter = w
		r.wroteAny = false
		exportLit, _ := json.Marshal(exportName)
		propsLit, _ := json.Marshal(propsJSON)
		script := globals.StartRenderFn + "(" + string(exportLit) + ", " + string(propsLit) + ")"
		if _, e := r.inner.Eval(script, mquickjs.EvalGlobal); e != nil {
			err = fmt.Errorf("moderncquickjs: start render: %w", e)
		}
	})
	if err != nil {
		return false, err
	}

	// Pull loop: each run is a separate event-loop task so the checkpoint
	// fires between iterations, allowing async components to advance.
	var wroteAny bool
	for {
		var done bool
		var iterWrote bool
		r.run(func() {
			r.wroteAny = false
			result, e := r.inner.Eval(globals.PullOnceFn+"()", mquickjs.EvalGlobal)
			if e != nil {
				err = fmt.Errorf("moderncquickjs: render: %w", e)
				return
			}
			done, _ = result.(bool)
			iterWrote = r.wroteAny
		})
		wroteAny = wroteAny || iterWrote
		if err != nil {
			break
		}
		if done {
			break
		}
		if !iterWrote {
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Cleanup: release writer reference and clear JS stream globals.
	r.run(func() {
		r.currentWriter = nil
		r.wroteAny = false
		evalOrErr(r.inner, globals.ClearRenderStreamFn+"()") //nolint:errcheck
	})

	return wroteAny, err
}

// clearState removes per-request data before the Runtime is returned to the pool.
func (r *Runtime) clearState() error {
	var runErr error
	r.run(func() {
		r.diFuncs = nil
		r.currentWriter = nil
		r.wroteAny = false
		_, runErr = r.inner.Eval(
			"globalThis."+globals.RequestContext+" = undefined; "+
				"Object.keys("+globals.BridgeObject+").forEach(function(k) { delete "+globals.BridgeObject+"[k]; });",
			mquickjs.EvalGlobal,
		)
	})
	return runErr
}

// Ensure *Runtime satisfies core.SSRRuntime at compile time.
var _ core.SSRRuntime = (*Runtime)(nil)

// ── StreamHandle ──────────────────────────────────────────────────────────────

type mqjsStreamHandle struct {
	exportName string
	propsJSON  string
}

func (h *mqjsStreamHandle) IsNil() bool { return h == nil }

// ── SSRPool adapter ───────────────────────────────────────────────────────────

type mqjsSSRPool struct{ pool *vmpool.Pool[*Runtime] }

func (p *mqjsSSRPool) Acquire() (core.SSRRuntime, error) { return p.pool.Acquire() }
func (p *mqjsSSRPool) Release(rt core.SSRRuntime) {
	if r, ok := rt.(*Runtime); ok {
		p.pool.Release(r)
	}
}

// ── vmPool ────────────────────────────────────────────────────────────────────

func newVMPool(serverBundle string, logger core.Logger) (*vmpool.Pool[*Runtime], error) {
	return vmpool.New(
		vmpool.Config{MinSize: 1, MaxSize: 64},
		func() (*Runtime, error) { return newRuntime(serverBundle, logger) },
		func(r *Runtime) { _ = r.clearState() },
	)
}

// Registered is true when the moderncquickjs build tag is active.
var Registered = true

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.JSEngine, error) {
			return NewEngine(), nil
		})
	})
}
