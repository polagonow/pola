//go:build v8go

// Package v8go provides a V8-backed JSEngine implementation for the Pola
// framework using rogchap.com/v8go (CGo-based).
//
// Build tag: v8go
package v8go

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/eventloop"
	"github.com/polagonow/pola/engine/polyfill"

	v8 "rogchap.com/v8go"
)

var (
	startRenderScriptTmpl = template.Must(template.New("startRender").Parse(
		"globalThis.{{.StreamHandle}} = {{.RenderFn}}({{.ExportNameLit}}, {{.PropsJSONLit}});"))
	pullScriptTmpl = template.Must(template.New("pull").Parse("{{.PullStreamFn}}({{.StreamHandle}})"))

	deleteKeyTmpl  = template.Must(template.New("deleteKey").Parse("delete {{.BridgeObject}}[{{.Key}}];"))
	clearStateTmpl = template.Must(template.New("clearState").Parse(
		"globalThis.{{.RequestContext}} = undefined; globalThis.{{.StreamHandle}} = undefined;"))
)

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine is a v8go-backed JSEngine. It holds the server bundle source for
// VM initialisation.
type Engine struct {
	source string
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// New creates an Engine from the given server bundle source.
func New(bundleSource string) *Engine {
	return &Engine{source: bundleSource}
}

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "v8go" }

// RequiredPolyfills implements core.JSEngine.
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
	return []core.PolyfillID{
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
	}
}

// NewRuntime implements core.JSEngine.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	return newV8Runtime(e.source, e.logger)
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a V8 isolate + context with a goroutine-pinned event loop.
type Runtime struct {
	iso     *v8.Isolate
	ctx     *v8.Context
	loop    *eventloop.EventLoop
	diKeys  []string // keys currently set on __DEPENDENCY_INJECTION__; cleared in clearState
}

func newV8Runtime(source string, logger core.Logger) (*Runtime, error) {
	iso := v8.NewIsolate()
	loop := eventloop.New(true)
	r := &Runtime{iso: iso, loop: loop}

	var initErr error
	loop.RunSync(func() {
		ctx := v8.NewContext(iso)
		r.ctx = ctx
		loop.SetCheckpoint(ctx.PerformMicrotaskCheckpoint)
		global := ctx.Global()

		global.Set("global", global)     //nolint:errcheck
		global.Set("globalThis", global) //nolint:errcheck

		// Console.
		if err := enableConsole(iso, ctx, logger); err != nil {
			initErr = fmt.Errorf("v8go: %w", err)
			return
		}

		// process / performance stubs.
		if _, err := ctx.RunScript(`
globalThis.process = { env: { NODE_ENV: "production" } };
globalThis.performance = { now: function() { return 0; } };
`, "env.js"); err != nil {
			initErr = fmt.Errorf("v8go: env globals: %w", err)
			return
		}

		// __DEPENDENCY_INJECTION__ bridge object.
		if _, err := ctx.RunScript("globalThis."+globals.BridgeObject+" = {};", "di.js"); err != nil {
			initErr = fmt.Errorf("v8go: di object: %w", err)
			return
		}

		// Polyfills.
		reg := polyfill.DefaultRegistry()
		for _, src := range reg.Get(
			polyfill.MicrotaskQueue,
			polyfill.TextEncoding,
			polyfill.MessageChannel,
			polyfill.ReadableStream,
			polyfill.AbortController,
			polyfill.WebpackRequire,
		) {
			if _, err := ctx.RunScript(src.Source, string(src.ID)+".js"); err != nil {
				initErr = fmt.Errorf("v8go: polyfill %s: %w", src.ID, err)
				return
			}
		}

		// Run server bundle.
		if _, err := ctx.RunScript(source, "bundle.js"); err != nil {
			initErr = fmt.Errorf("v8go: run bundle: %w", err)
			return
		}
	})

	if initErr != nil {
		loop.Stop()
		iso.Dispose()
		return nil, initErr
	}
	return r, nil
}

// run executes fn synchronously on the event loop.
func (r *Runtime) run(fn func() error) error {
	var runErr error
	r.loop.RunSync(func() {
		runErr = fn()
	})
	return runErr
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	err := r.run(func() error {
		v, err := r.ctx.RunScript(script, "eval.js")
		if err != nil {
			return err
		}
		result = v8ValueToGo(v)
		return nil
	})
	return result, err
}

// Call implements core.JSRuntime.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("v8go: marshal args: %w", err)
	}
	script := fmt.Sprintf("(function(){ var __args = %s; return %s.apply(null, __args); })()", string(argsJSON), fn)
	return r.Eval(script)
}

// Set implements core.JSRuntime.
func (r *Runtime) Set(name string, value any) error {
	return r.run(func() error {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("v8go: marshal value for %s: %w", name, err)
		}
		_, err = r.ctx.RunScript("globalThis."+name+" = "+string(data)+";", "set.js")
		return err
	})
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.loop.Stop()
	r.iso.Dispose()
}

// SetRequestContext injects per-request data as __REQUEST__.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	data, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("v8go: marshal request context: %w", err)
	}
	return r.run(func() error {
		_, err := r.ctx.RunScript(
			"globalThis."+globals.RequestContext+" = "+string(data)+";",
			"request.js",
		)
		return err
	})
}

// SetDependencyInjection installs async bridge functions on __DEPENDENCY_INJECTION__.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	return r.run(func() error {
		// Clear previous keys.
		for _, key := range r.diKeys {
			var delBuf strings.Builder
			_ = deleteKeyTmpl.Execute(&delBuf, struct{ BridgeObject, Key string }{
				globals.BridgeObject, fmt.Sprintf("%q", key),
			})
			r.ctx.RunScript(delBuf.String(), "di_clear.js") //nolint:errcheck
		}
		r.diKeys = r.diKeys[:0]

		if len(funcs) == 0 {
			return nil
		}

		jsiVal, err := r.ctx.Global().Get(globals.BridgeObject)
		if err != nil {
			return fmt.Errorf("v8go: get bridge object: %w", err)
		}
		jsiObj := jsiVal.Object()

		for name, fn := range funcs {
			fn := fn // capture
			ft := v8.NewFunctionTemplate(r.iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
				args := exportV8Args(info.Args())
				resolver, err := v8.NewPromiseResolver(info.Context())
				if err != nil {
					return nil
				}
				go func() {
					result, err := fn(args)
					r.loop.RunAsync(func() {
						if err != nil {
							errVal, _ := v8.NewValue(r.iso, err.Error())
							resolver.Reject(errVal) //nolint:errcheck
						} else {
							val, convErr := goToV8Value(r.iso, r.ctx, result)
							if convErr != nil {
								errVal, _ := v8.NewValue(r.iso, convErr.Error())
								resolver.Reject(errVal) //nolint:errcheck
							} else {
								resolver.Resolve(val) //nolint:errcheck
							}
						}
					})
				}()
				return resolver.GetPromise().Value
			})
			jsiObj.Set(name, ft.GetFunction(r.ctx)) //nolint:errcheck
			r.diKeys = append(r.diKeys, name)
		}
		return nil
	})
}

// RenderSession is a zero-size token; the stream is stored in the __pola_stream__ global.
type RenderSession struct{}

// StartRender calls __render__(exportName, propsJSON) storing the stream in the
// global __pola_stream__ for DrainStream to consume.
func (r *Runtime) StartRender(exportName, propsJSON string) (RenderSession, error) {
	var sess RenderSession
	err := r.run(func() error {
		exportNameLit, _ := json.Marshal(exportName)
		propsJSONLit, _ := json.Marshal(propsJSON)
		var scriptBuf strings.Builder
		_ = startRenderScriptTmpl.Execute(&scriptBuf, struct {
			StreamHandle, RenderFn         string
			ExportNameLit, PropsJSONLit string
		}{globals.StreamHandle, globals.RenderFn, string(exportNameLit), string(propsJSONLit)})
		_, err := r.ctx.RunScript(scriptBuf.String(), "render.js")
		return err
	})
	return sess, err
}

// DrainStream polls the stream until done, writing decoded chunks to w.
func (r *Runtime) DrainStream(_ RenderSession, w core.StreamWriter) (bool, error) {
	var wroteAny bool
	for {
		var done, noChunks bool
		if runErr := r.run(func() error {
			var pullBuf strings.Builder
			_ = pullScriptTmpl.Execute(&pullBuf, struct{ PullStreamFn, StreamHandle string }{
				globals.PullStreamFn, globals.StreamHandle,
			})
			result, err := r.ctx.RunScript(pullBuf.String(), "pull.js")
			if err != nil {
				return fmt.Errorf("v8go: pull stream: %w", err)
			}
			resultObj := result.Object()

			chunksVal, err := resultObj.Get("chunks")
			if err != nil {
				return fmt.Errorf("v8go: get chunks: %w", err)
			}
			doneVal, err := resultObj.Get("done")
			if err != nil {
				return fmt.Errorf("v8go: get done: %w", err)
			}
			done = doneVal.Boolean()

			chunksObj := chunksVal.Object()
			lenVal, _ := chunksObj.Get("length")
			chunksLen := int(lenVal.Integer())

			if chunksLen > 0 {
				var sb strings.Builder
				for i := 0; i < chunksLen; i++ {
					chunkVal, err := chunksObj.GetIdx(uint32(i))
					if err != nil {
						continue
					}
					if chunkVal.IsUint8Array() {
						sb.Write(readUint8Array(chunkVal.Object()))
					} else {
						sb.WriteString(chunkVal.String())
					}
				}
				w.WriteRaw([]byte(sb.String())) //nolint:errcheck
				w.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			return nil
		}); runErr != nil {
			return wroteAny, runErr
		}
		if done {
			break
		}
		if noChunks {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return wroteAny, nil
}

func (r *Runtime) clearState() error {
	return r.run(func() error {
		for _, key := range r.diKeys {
			var delBuf strings.Builder
			_ = deleteKeyTmpl.Execute(&delBuf, struct{ BridgeObject, Key string }{
				globals.BridgeObject, fmt.Sprintf("%q", key),
			})
			r.ctx.RunScript(delBuf.String(), "di_clear.js") //nolint:errcheck
		}
		r.diKeys = r.diKeys[:0]
		var clearBuf strings.Builder
		_ = clearStateTmpl.Execute(&clearBuf, struct{ RequestContext, StreamHandle string }{
			globals.RequestContext, globals.StreamHandle,
		})
		r.ctx.RunScript(clearBuf.String(), "clear.js") //nolint:errcheck
		return nil
	})
}

// ── VMPool ────────────────────────────────────────────────────────────────────

// VMPool manages a pool of pre-warmed V8 Runtimes.
type VMPool struct {
	pool   sync.Pool
	source string
	logger core.Logger
}

// NewVMPool creates a pool backed by the given server-bundle source.
// One Runtime is eagerly created to surface startup errors immediately.
func NewVMPool(serverBundle string, logger core.Logger) (*VMPool, error) {
	p := &VMPool{source: serverBundle, logger: logger}
	p.pool = sync.Pool{
		New: func() any {
			r, err := newV8Runtime(serverBundle, logger)
			if err != nil {
				panic(fmt.Sprintf("v8go: pool create: %v", err))
			}
			return r
		},
	}
	r := p.pool.New()
	p.pool.Put(r)
	return p, nil
}

// Acquire returns a Runtime from the pool.
func (p *VMPool) Acquire() *Runtime { return p.pool.Get().(*Runtime) }

// Release clears per-request state and returns the Runtime to the pool.
func (p *VMPool) Release(r *Runtime) {
	_ = r.clearState()
	p.pool.Put(r)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func enableConsole(iso *v8.Isolate, ctx *v8.Context, logger core.Logger) error {
	const src = `
globalThis.console = {
	log:   function() { __v8_log__("LOG",  Array.prototype.slice.call(arguments)); },
	warn:  function() { __v8_log__("WARN", Array.prototype.slice.call(arguments)); },
	error: function() { __v8_log__("ERR",  Array.prototype.slice.call(arguments)); },
	info:  function() { __v8_log__("INFO", Array.prototype.slice.call(arguments)); },
};`
	logFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) < 2 {
			return nil
		}
		level := args[0].String()
		arr := args[1].Object()
		lenVal, _ := arr.Get("length")
		n := int(lenVal.Integer())
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			v, _ := arr.GetIdx(uint32(i))
			parts[i] = v.String()
		}
		if logger != nil {
			logger.Debug("js console", "engine", "v8go", "level", level, "output", strings.Join(parts, " "))
		}
		return nil
	})
	if err := ctx.Global().Set("__v8_log__", logFT.GetFunction(ctx)); err != nil {
		return fmt.Errorf("console: set __v8_log__: %w", err)
	}
	if _, err := ctx.RunScript(src, "console.js"); err != nil {
		return fmt.Errorf("console: %w", err)
	}
	return nil
}

func goToV8Value(iso *v8.Isolate, ctx *v8.Context, v any) (*v8.Value, error) {
	switch t := v.(type) {
	case nil:
		return v8.Null(iso), nil
	case bool:
		return v8.NewValue(iso, t)
	case string:
		return v8.NewValue(iso, t)
	case int:
		return v8.NewValue(iso, int32(t))
	case int32:
		return v8.NewValue(iso, t)
	case int64:
		return v8.NewValue(iso, t)
	case uint32:
		return v8.NewValue(iso, t)
	case uint64:
		return v8.NewValue(iso, t)
	case float32:
		return v8.NewValue(iso, float64(t))
	case float64:
		return v8.NewValue(iso, t)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("v8 value: json marshal: %w", err)
		}
		return ctx.RunScript("("+string(data)+")", "value.js")
	}
}

func v8ValueToGo(v *v8.Value) any {
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	if v.IsBoolean() {
		return v.Boolean()
	}
	if v.IsNumber() {
		n := v.Number()
		if n == float64(int64(n)) {
			return int64(n)
		}
		return n
	}
	if v.IsString() {
		return v.String()
	}
	return v.String()
}

func exportV8Args(args []*v8.Value) []any {
	out := make([]any, len(args))
	for i, v := range args {
		out[i] = v8ValueToGo(v)
	}
	return out
}

func readUint8Array(obj *v8.Object) []byte {
	lenVal, err := obj.Get("length")
	if err != nil {
		return nil
	}
	n := int(lenVal.Integer())
	data := make([]byte, n)
	for i := 0; i < n; i++ {
		v, err := obj.GetIdx(uint32(i))
		if err != nil {
			continue
		}
		data[i] = byte(v.Integer())
	}
	return data
}
