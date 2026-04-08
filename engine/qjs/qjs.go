// Package qjs provides a QuickJS-backed JSEngine implementation for the Pola
// framework using github.com/fastschema/qjs (CGo-free, Wazero-based).
//
// Async bridge functions are created with ctx.Function(fn) — the goroutine
// spawned by each bridge call blocks until the result is available then returns
// the JSON-encoded value. The render is driven by promise.Await() on the
// __renderAsync__ promise.
//
// Build tag: qjs
package qjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/polyfill"

	qjslib "github.com/fastschema/qjs"
)

// renderAsyncFn is the async JS helper that drives the render loop in qjs.
const renderAsyncFn = "__renderAsync__"

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
		panic("qjs: renderAsync template: " + err.Error())
	}
	renderAsyncJS = b.String()
}

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine is a qjs-backed JSEngine.
type Engine struct {
	src    string
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// New creates an Engine from the given server bundle source.
func New(bundleSource string) *Engine {
	return &Engine{src: bundleSource}
}

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "qjs" }

// RequiredPolyfills implements core.JSEngine.
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
	return []core.PolyfillID{
		polyfill.NodeGlobals,
		polyfill.Promise,
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
		polyfill.ConsoleBridge,
	}
}

// NewRuntime implements core.JSEngine.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	return newRuntime(e.src, e.logger)
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a qjs runtime + context. All JS operations are serialised via mu.
type Runtime struct {
	rt  *qjslib.Runtime
	ctx *qjslib.Context
	mu  sync.Mutex
	src string
}

func newRuntime(src string, logger core.Logger) (*Runtime, error) {
	rt, err := qjslib.New()
	if err != nil {
		return nil, fmt.Errorf("qjs: new runtime: %w", err)
	}
	ctx := rt.Context()

	r := &Runtime{rt: rt, ctx: ctx, src: src}
	var initErr error

	r.run(func() {
		// Install __pola_log__ Go callback (wired to logger) before ConsoleBridge polyfill.
		ctx.SetFunc(globals.PolaLogFn, func(this *qjslib.This) (*qjslib.Value, error) {
			args := this.Args()
			if len(args) >= 2 {
				polyfill.LogAtLevel(logger, "qjs", args[0].String(), args[1].String())
			}
			return this.Context().NewUndefined(), nil
		})

		if err := evalOrErr(ctx, "globalThis."+globals.BridgeObject+" = {};", "di.js"); err != nil {
			initErr = err
			return
		}

		// Install polyfills (NodeGlobals + ConsoleBridge must come before bundle).
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
				initErr = fmt.Errorf("qjs: polyfill %s: %w", pf.ID, err)
				return
			}
		}

		// Run server bundle.
		if err := evalOrErr(ctx, src, "bundle.js"); err != nil {
			initErr = fmt.Errorf("qjs: run bundle: %w", err)
			return
		}

		// Install async render helper (depends on bundle's __render__).
		if err := evalOrErr(ctx, renderAsyncJS, "render_async.js"); err != nil {
			initErr = fmt.Errorf("qjs: render async: %w", err)
		}
	})

	if initErr != nil {
		rt.Close()
		return nil, initErr
	}
	return r, nil
}

func (r *Runtime) run(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn()
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	var runErr error
	r.run(func() {
		val, err := r.ctx.Eval("eval.js", qjslib.Code(script))
		if err != nil {
			runErr = err
			return
		}
		defer val.Free()
		result = qjsValueToGo(val)
	})
	return result, runErr
}

// Call implements core.JSRuntime.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("qjs: marshal args: %w", err)
	}
	script := fmt.Sprintf("(function(){ var __args = %s; return %s.apply(null, __args); })()", string(argsJSON), fn)
	return r.Eval(script)
}

// Set implements core.JSRuntime.
func (r *Runtime) Set(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("qjs: marshal value for %s: %w", name, err)
	}
	var runErr error
	r.run(func() {
		runErr = evalOrErr(r.ctx, "globalThis."+name+" = "+string(data)+";", "set.js")
	})
	return runErr
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.run(func() { r.rt.Close() })
}

// SetRequestContext injects per-request data as __REQUEST__.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("qjs: marshal request context: %w", err)
	}
	var runErr error
	r.run(func() {
		val := r.ctx.ParseJSON(string(b))
		r.ctx.Global().SetPropertyStr(globals.RequestContext, val)
	})
	return runErr
}

// bridgeResult carries the outcome of an async Go bridge call.
type bridgeResult struct {
	val any
	err error
}

// SetDependencyInjection installs bridge functions on __DEPENDENCY_INJECTION__.
// Bridge functions block the host-function call while the Go goroutine runs.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	r.run(func() {
		// Clear existing keys.
		evalOrErr(r.ctx, //nolint:errcheck
			"Object.keys("+globals.BridgeObject+").forEach(function(k) { delete "+globals.BridgeObject+"[k]; });",
			"clear_di.js")

		jsi := r.ctx.Global().GetPropertyStr(globals.BridgeObject)
		defer jsi.Free()

		for name, fn := range funcs {
			fn := fn // capture
			jsFn := r.ctx.Function(func(this *qjslib.This) (*qjslib.Value, error) {
				goArgs := exportArgs(this.Args())
				ch := make(chan bridgeResult, 1)
				go func() {
					v, e := fn(goArgs)
					ch <- bridgeResult{v, e}
				}()
				res := <-ch
				if res.err != nil {
					return nil, res.err
				}
				if res.val == nil {
					return this.Context().NewNull(), nil
				}
				b, marshalErr := json.Marshal(res.val)
				if marshalErr != nil {
					return nil, marshalErr
				}
				return this.Context().ParseJSON(string(b)), nil
			})
			jsi.SetPropertyStr(name, jsFn)
		}
	})
	return nil
}

// RenderSession holds parameters for a single render.
type RenderSession struct {
	ExportName string
	PropsJSON  string
}

// StartRender stores render parameters for DrainStream.
func (r *Runtime) StartRender(exportName, propsJSON string) (RenderSession, error) {
	return RenderSession{ExportName: exportName, PropsJSON: propsJSON}, nil
}

// DrainStream installs a per-request __outputChunk__ Go function, then calls
// __renderAsync__(exportName, propsJSON) and awaits the returned Promise.
func (r *Runtime) DrainStream(sess RenderSession, w core.StreamWriter) (bool, error) {
	var wroteAny bool
	var runErr error

	r.run(func() {
		// Install the per-request output sink.
		r.ctx.SetFunc(globals.OutputChunk, func(this *qjslib.This) (*qjslib.Value, error) {
			args := this.Args()
			if len(args) > 0 && args[0].IsString() {
				if chunk := args[0].String(); len(chunk) > 0 {
					w.WriteRaw([]byte(chunk)) //nolint:errcheck
					w.Flush()
					wroteAny = true
				}
			}
			return this.Context().NewUndefined(), nil
		})

		exportLit, _ := json.Marshal(sess.ExportName)
		propsLit, _ := json.Marshal(sess.PropsJSON)
		script := fmt.Sprintf("%s(%s, %s)", renderAsyncFn, string(exportLit), string(propsLit))

		promise, evalErr := r.ctx.Eval("render.js", qjslib.Code(script))
		if evalErr != nil {
			runErr = fmt.Errorf("qjs: render eval: %w", evalErr)
			return
		}
		defer promise.Free()

		result, awaitErr := promise.Await()
		if result != nil {
			defer result.Free()
		}
		if awaitErr != nil {
			runErr = fmt.Errorf("qjs: render await: %w", awaitErr)
			return
		}

		_, _ = r.ctx.Eval("clear_output.js", qjslib.Code(globals.OutputChunk+" = undefined;"))
	})

	return wroteAny, runErr
}

func (r *Runtime) clearState() {
	r.run(func() {
		evalOrErr(r.ctx, //nolint:errcheck
			globals.RequestContext+" = undefined; "+
				globals.StreamHandle+" = undefined; "+
				globals.OutputChunk+" = undefined; "+
				"globalThis."+globals.BridgeObject+" = {};",
			"clear_state.js")
	})
}

// ── VMPool ────────────────────────────────────────────────────────────────────

// VMPool manages a pool of pre-warmed qjs Runtimes.
type VMPool struct {
	pool   sync.Pool
	src    string
	logger core.Logger
}

// NewVMPool creates a pool backed by the given server-bundle source.
func NewVMPool(serverBundle string, logger core.Logger) (*VMPool, error) {
	p := &VMPool{src: serverBundle, logger: logger}
	p.pool = sync.Pool{
		New: func() any {
			r, err := newRuntime(serverBundle, logger)
			if err != nil {
				panic(fmt.Sprintf("qjs: pool create: %v", err))
			}
			return r
		},
	}
	// Eagerly create one Runtime to catch startup errors immediately.
	r, err := newRuntime(serverBundle, logger)
	if err != nil {
		return nil, fmt.Errorf("qjs: pool create: %w", err)
	}
	p.pool.Put(r)
	return p, nil
}

// Acquire returns a Runtime from the pool. If the pool needs to create a new
// Runtime and that fails, the error is returned instead of panicking.
func (p *VMPool) Acquire() (rt *Runtime, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return p.pool.Get().(*Runtime), nil
}

// Release clears per-request state and returns the Runtime to the pool.
func (p *VMPool) Release(r *Runtime) {
	r.clearState()
	p.pool.Put(r)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func evalOrErr(ctx *qjslib.Context, src, name string) error {
	val, err := ctx.Eval(name, qjslib.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

func exportArgs(args []*qjslib.Value) []any {
	out := make([]any, len(args))
	for i, v := range args {
		switch {
		case v.IsNull() || v.IsUndefined():
			out[i] = nil
		case v.IsString():
			out[i] = v.String()
		case v.IsNumber():
			out[i] = v.Float64()
		case v.IsBool():
			out[i] = v.Bool()
		default:
			out[i] = v.String()
		}
	}
	return out
}

func qjsValueToGo(v *qjslib.Value) any {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	if v.IsBool() {
		return v.Bool()
	}
	if v.IsNumber() {
		return v.Float64()
	}
	return v.String()
}
