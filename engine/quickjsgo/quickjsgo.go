// Package quickjsgo provides a QuickJS-backed JSEngine for the Pola framework
// using github.com/buke/quickjs-go.
//
// Async bridge functions resolve via ctx.Schedule and the entire render is
// driven by ctx.Await on a JS async function. This handles both native Promise
// microtasks and Go-scheduled callbacks in a single cooperative polling loop.
//
// Note: unlike the streaming Goja/Sobek/v8go implementations, quickjsgo
// collects all rendered output in JS before writing to the StreamWriter.
//
// Build tag: quickjsgo
package quickjsgo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	quickjs "github.com/buke/quickjs-go"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/vmpool"
)

// renderAsyncFn is the async JS helper that drives the render loop in quickjsgo.
const renderAsyncFn = "__renderAsync__"

// ── JS templates ──────────────────────────────────────────────────────────────

var (
	renderAsyncJSTmpl = template.Must(template.New("renderAsync").Parse(`
globalThis.{{.RenderAsyncFn}} = async function(exportName, propsJSON) {
	var stream = {{.RenderFn}}(exportName, propsJSON);
	var dec = new TextDecoder();
	while (true) {
		{{.DrainMicrotasksFn}}();
		var result = {{.PullStreamFn}}(stream);
		for (var i = 0; i < result.chunks.length; i++) {
			{{.OutputChunk}}(dec.decode(result.chunks[i]));
		}
		if (result.done) return;
		// Yield so ctx.Await() can run pending JS microtasks and Go callbacks.
		await Promise.resolve();
	}
};
`))

	renderAsyncJS string
)

func init() {
	var b strings.Builder
	if err := renderAsyncJSTmpl.Execute(&b, struct {
		RenderAsyncFn, RenderFn, DrainMicrotasksFn, PullStreamFn, OutputChunk string
	}{
		renderAsyncFn,
		globals.RenderFn,
		globals.DrainMicrotasksFn,
		globals.PullStreamFn,
		globals.OutputChunk,
	}); err != nil {
		panic("quickjsgo: renderAsync template: " + err.Error())
	}
	renderAsyncJS = b.String()
}

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine is a quickjsgo-backed JSEngine. It implements core.JSEngine and
// core.SSRPoolFactory.
type Engine struct {
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// NewEngine returns a stateless Engine (no bundle pre-compilation needed).
func NewEngine() *Engine { return &Engine{} }

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "quickjsgo" }

// RequiredPolyfills implements core.JSEngine.
// quickjsgo needs all polyfills including Promise.
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
	return []core.PolyfillID{
		polyfill.NodeGlobals,
		polyfill.ConsoleBridge,
		polyfill.Promise,
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
	}
}

// NewRuntime implements core.JSEngine. Returns an error — use NewSSRPool for
// bundle-backed runtimes; the SSR pipeline calls that instead.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	return nil, fmt.Errorf("quickjsgo: use NewSSRPool to create bundle-backed runtimes")
}

// NewSSRPool implements core.SSRPoolFactory.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
	pool, err := newVMPool(string(bundle), e.logger)
	if err != nil {
		return nil, err
	}
	return &qjsgoSSRPool{pool: pool}, nil
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a quickjs-go runtime + context. All JS operations are
// serialised via the mutex.
type Runtime struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
	mu  sync.Mutex
	jsi *quickjs.Value // persistent __DEPENDENCY_INJECTION__ object reference
}

func (r *Runtime) run(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn()
}

func evalOrErr(ctx *quickjs.Context, src, name string) error {
	ret := ctx.Eval(src, quickjs.EvalFileName(name))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("%s: %w", name, ctx.Exception())
	}
	return nil
}

func setGlobal(ctx *quickjs.Context, name string, val *quickjs.Value) {
	g := ctx.Globals()
	g.Set(name, val) //nolint:errcheck
}

func newRuntime(src string, logger core.Logger) (*Runtime, error) {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()

	r := &Runtime{rt: rt, ctx: ctx}
	var initErr error

	r.run(func() {
		// Install __pola_log__ Go callback (wired to logger) before ConsoleBridge polyfill.
		logFn := ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) >= 2 {
				polyfill.LogAtLevel(logger, "quickjsgo", args[0].String(), args[1].String())
			}
			return qCtx.NewUndefined()
		})
		setGlobal(ctx, globals.PolaLogFn, logFn)

		// __DEPENDENCY_INJECTION__ placeholder.
		if err := evalOrErr(ctx, "globalThis."+globals.BridgeObject+" = {};", "di.js"); err != nil {
			initErr = err
			return
		}

		// Keep a persistent reference for SetDependencyInjection.
		jsiVal := ctx.Eval(globals.BridgeObject, quickjs.EvalFileName("get_di.js"))
		if jsiVal.IsException() {
			initErr = fmt.Errorf("quickjsgo: get %s: %w", globals.BridgeObject, ctx.Exception())
			jsiVal.Free()
			return
		}
		r.jsi = jsiVal

		// Polyfills (NodeGlobals + ConsoleBridge must come before bundle).
		reg := polyfill.DefaultRegistry()
		for _, pf := range reg.Get(
			polyfill.NodeGlobals,
			polyfill.ConsoleBridge,
			polyfill.Promise,
			polyfill.MicrotaskQueue,
			polyfill.TextEncoding,
			polyfill.MessageChannel,
			polyfill.ReadableStream,
			polyfill.AbortController,
			polyfill.WebpackRequire,
		) {
			if err := evalOrErr(ctx, pf.Source, string(pf.ID)+".js"); err != nil {
				initErr = fmt.Errorf("quickjsgo: polyfill %s: %w", pf.ID, err)
				return
			}
		}

		// Server bundle.
		if err := evalOrErr(ctx, src, "bundle.js"); err != nil {
			initErr = fmt.Errorf("quickjsgo: run bundle: %w", err)
			return
		}

		// Async render helper.
		if err := evalOrErr(ctx, renderAsyncJS, "render_async.js"); err != nil {
			initErr = fmt.Errorf("quickjsgo: render async: %w", err)
		}
	})

	if initErr != nil {
		ctx.Close()
		rt.Close()
		return nil, initErr
	}
	return r, nil
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	var runErr error
	r.run(func() {
		ret := r.ctx.Eval(script, quickjs.EvalFileName("eval.js"))
		defer ret.Free()
		if ret.IsException() {
			runErr = r.ctx.Exception()
			return
		}
		result = exportValue(ret)
	})
	return result, runErr
}

// Call implements core.JSRuntime.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("quickjsgo: marshal args: %w", err)
	}
	script := fmt.Sprintf("(function(){ var __args = %s; return %s.apply(null, __args); })()", string(argsJSON), fn)
	return r.Eval(script)
}

// Set implements core.JSRuntime.
func (r *Runtime) Set(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("quickjsgo: marshal value for %s: %w", name, err)
	}
	var runErr error
	r.run(func() {
		runErr = evalOrErr(r.ctx, "globalThis."+name+" = ("+string(data)+");", "set.js")
	})
	return runErr
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.run(func() {
		if r.jsi != nil {
			r.jsi.Free()
			r.jsi = nil
		}
		r.ctx.Close()
		r.rt.Close()
	})
}

// SetRequestContext implements core.SSRRuntime.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	var runErr error
	r.run(func() {
		val, err := r.ctx.Marshal(ctx)
		if err != nil {
			runErr = fmt.Errorf("quickjsgo: marshal request context: %w", err)
			return
		}
		g := r.ctx.Globals()
		g.Set(globals.RequestContext, val) //nolint:errcheck
	})
	return runErr
}

// SetDependencyInjection installs async bridge functions on __DEPENDENCY_INJECTION__.
// Each function returns a Promise; results are delivered via ctx.Schedule.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	r.run(func() {
		// Clear existing keys.
		clearRet := r.ctx.Eval(
			"Object.keys("+globals.BridgeObject+").forEach(function(k) { delete "+globals.BridgeObject+"[k]; });",
			quickjs.EvalFileName("clear_di.js"),
		)
		clearRet.Free()

		for name, fn := range funcs {
			fn := fn // capture
			bridgeFn := r.ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
				goArgs := exportArgs(args)

				var resolveFn, rejectFn func(*quickjs.Value)
				p := qCtx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
					resolveFn = resolve
					rejectFn = reject
				})

				go func() {
					result, err := fn(goArgs)
					qCtx.Schedule(func(innerCtx *quickjs.Context) {
						if err != nil {
							errVal := innerCtx.NewString(err.Error())
							rejectFn(errVal)
							errVal.Free()
						} else {
							val, marshalErr := innerCtx.Marshal(result)
							if marshalErr != nil {
								errVal := innerCtx.NewString(marshalErr.Error())
								rejectFn(errVal)
								errVal.Free()
								return
							}
							resolveFn(val)
							val.Free()
						}
					})
				}()

				return p
			})
			r.jsi.Set(name, bridgeFn) //nolint:errcheck
			// Do NOT free bridgeFn: Set transfers ownership.
		}
	})
	return nil
}

// CallRenderFunction implements core.SSRRuntime. Stores the render parameters;
// the actual JS execution happens in DrainStream.
func (r *Runtime) CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error) {
	return &qjsgoStreamHandle{exportName: exportName, propsJSON: propsJSON}, nil
}

// DrainStream implements core.SSRRuntime.
func (r *Runtime) DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error) {
	h, ok := handle.(*qjsgoStreamHandle)
	if !ok {
		return false, fmt.Errorf("quickjsgo: DrainStream: unexpected handle type %T", handle)
	}
	return r.drainStream(h.exportName, h.propsJSON, w)
}

func (r *Runtime) drainStream(exportName, propsJSON string, w core.StreamWriter) (bool, error) {
	var wroteAny bool
	var runErr error

	r.run(func() {
		// Install per-request __outputChunk__. Called synchronously from JS
		// during ctx.Await, so chunks are flushed to w incrementally.
		outputFn := r.ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) > 0 && args[0].IsString() {
				if chunk := args[0].String(); len(chunk) > 0 {
					w.WriteRaw([]byte(chunk)) //nolint:errcheck
					w.Flush()
					wroteAny = true
				}
			}
			return qCtx.NewUndefined()
		})
		g := r.ctx.Globals()
		g.Set(globals.OutputChunk, outputFn) // ownership transferred; do NOT free outputFn or g

		exportLit, _ := json.Marshal(exportName)
		propsLit, _ := json.Marshal(propsJSON)
		script := renderAsyncFn + "(" + string(exportLit) + ", " + string(propsLit) + ")"

		promise := r.ctx.Eval(script, quickjs.EvalFileName("render.js"))
		defer promise.Free()
		if promise.IsException() {
			runErr = fmt.Errorf("quickjsgo: render eval: %w", r.ctx.Exception())
			return
		}

		// ctx.Await drives the event loop: runs JS_ExecutePendingJob (native
		// Promise continuations) and ProcessJobs (Go-scheduled callbacks).
		result := r.ctx.Await(promise)
		defer result.Free()
		if result.IsException() {
			runErr = fmt.Errorf("quickjsgo: render await: %w", r.ctx.Exception())
			return
		}

		// Clear the per-request sink.
		clearRet := r.ctx.Eval(globals.OutputChunk+" = undefined;", quickjs.EvalFileName("clear_output.js"))
		clearRet.Free()
	})

	return wroteAny, runErr
}

// clearState removes per-request data before the Runtime is returned to the pool.
func (r *Runtime) clearState() error {
	r.run(func() {
		clearRet := r.ctx.Eval(
			globals.RequestContext+" = undefined; "+
				globals.StreamHandle+" = undefined; "+
				globals.OutputChunk+" = undefined; "+
				"globalThis."+globals.BridgeObject+" = {};",
			quickjs.EvalFileName("clear_state.js"),
		)
		clearRet.Free()
	})
	return nil
}

// Ensure *Runtime satisfies core.SSRRuntime at compile time.
var _ core.SSRRuntime = (*Runtime)(nil)

// ── StreamHandle ──────────────────────────────────────────────────────────────

type qjsgoStreamHandle struct {
	exportName string
	propsJSON  string
}

func (h *qjsgoStreamHandle) IsNil() bool { return h == nil }

// ── SSRPool adapter ───────────────────────────────────────────────────────────

type qjsgoSSRPool struct{ pool *vmpool.Pool[*Runtime] }

func (p *qjsgoSSRPool) Acquire() (core.SSRRuntime, error) { return p.pool.Acquire() }
func (p *qjsgoSSRPool) Release(rt core.SSRRuntime) {
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

// ── helpers ───────────────────────────────────────────────────────────────────

func exportArgs(args []*quickjs.Value) []any {
	out := make([]any, len(args))
	for i, v := range args {
		switch {
		case v.IsNull() || v.IsUndefined():
			out[i] = nil
		case v.IsString():
			out[i] = v.String()
		case v.IsNumber():
			out[i] = v.ToFloat64()
		case v.IsBool():
			out[i] = v.ToBool()
		default:
			out[i] = v.String()
		}
	}
	return out
}

func exportValue(v *quickjs.Value) any {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	if v.IsBool() {
		return v.ToBool()
	}
	if v.IsNumber() {
		return v.ToFloat64()
	}
	return v.String()
}

// Registered is true when the quickjsgo build tag is active.
var Registered = true
