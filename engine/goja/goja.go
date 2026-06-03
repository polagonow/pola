// Package goja provides a Goja-backed JSEngine implementation for the Pola
// framework.
//
// Each Runtime is backed by a goja_nodejs EventLoop. The loop gives us real
// setTimeout/setInterval scheduling and — crucially — proper microtask
// flushing between ticks so that async server components (async/await)
// resolve correctly inside renderToReadableStream.
//
// Build tag: goja
package goja

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/vmpool"

	gojalib "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine compiles a server bundle once and creates lightweight Runtimes from it.
// It implements core.JSEngine and core.SSRPoolFactory.
type Engine struct {
	prog   *gojalib.Program // compiled server bundle; nil until WithBundle is called
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// NewEngine returns an Engine without a pre-compiled bundle.
// The bundle is compiled later via NewSSRPool (called by the pipeline).
func NewEngine() *Engine { return &Engine{} }

// New compiles bundleSource and returns a ready Engine.
func New(bundleSource string) (*Engine, error) {
	prog, err := gojalib.Compile("bundle.js", bundleSource, false)
	if err != nil {
		return nil, fmt.Errorf("goja: compile: %w", err)
	}
	return &Engine{prog: prog}, nil
}

// WithBundle compiles the given server bundle source and returns a new Engine
// with that bundle loaded. It is the preferred way to attach a bundle to an
// engine factory returned from init().
func (e *Engine) WithBundle(bundleSource string) (*Engine, error) {
	return New(bundleSource)
}

// NewSSRPool implements core.SSRPoolFactory. It compiles the server bundle and
// returns a pool of goja Runtimes that implement core.SSRRuntime.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
	pool, err := NewVMPool(string(bundle), e.logger)
	if err != nil {
		return nil, err
	}
	return &gojaSSRPool{pool: pool.pool, logger: e.logger}, nil
}

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "goja" }

// RequiredPolyfills implements core.JSEngine.
// Goja needs all polyfills except Promise (Goja has native Promises).
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
	return []core.PolyfillID{
		polyfill.ReadableStream,
		polyfill.MessageChannel,
		polyfill.AbortController,
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.WebpackRequire,
	}
}

// NewRuntime implements core.JSEngine.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
	if e.prog == nil {
		return nil, fmt.Errorf("goja: engine has no compiled bundle — call WithBundle first")
	}
	return newRuntime(e.prog, e.logger)
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a goja_nodejs EventLoop with the pre-loaded server bundle.
// All Goja operations run on the event loop goroutine via loop.Run.
// It implements core.JSRuntime plus Goja-specific rendering helpers.
type Runtime struct {
	loop *eventloop.EventLoop
	rt   *gojalib.Runtime // captured during init; valid for Runtime lifetime
	di   *gojalib.Object  // persistent __DEPENDENCY_INJECTION__ object
}

// newRuntime creates a fresh EventLoop, installs globals + polyfills, runs the bundle.
func newRuntime(prog *gojalib.Program, logger core.Logger) (*Runtime, error) {
	loop := eventloop.NewEventLoop(eventloop.EnableConsole(false))

	r := &Runtime{loop: loop}
	err := r.run(func(rt *gojalib.Runtime) error {
		r.rt = rt
		r.di = rt.NewObject()
		rt.Set(globals.BridgeObject, r.di)      //nolint:errcheck
		rt.Set("globalThis", rt.GlobalObject()) //nolint:errcheck

		// Install __pola_log__ so ConsoleBridge polyfill can wire console.* to the logger.
		rt.Set(globals.PolaLogFn, func(c gojalib.FunctionCall) gojalib.Value { //nolint:errcheck
			if len(c.Arguments) >= 2 {
				polyfill.LogAtLevel(logger, "goja", c.Arguments[0].String(), c.Arguments[1].String())
			}
			return gojalib.Undefined()
		})

		// Install polyfills via the default registry.
		// NodeGlobals and ConsoleBridge must come first (before bundle).
		reg := polyfill.DefaultRegistry()
		for _, src := range reg.Get(
			polyfill.NodeGlobals,
			polyfill.ConsoleBridge,
			polyfill.MicrotaskQueue,
			polyfill.TextEncoding,
			polyfill.MessageChannel,
			polyfill.ReadableStream,
			polyfill.AbortController,
			polyfill.WebpackRequire,
		) {
			if _, err := rt.RunString(src.Source); err != nil {
				return fmt.Errorf("goja: polyfill %s: %w", src.ID, err)
			}
		}

		if _, err := rt.RunProgram(prog); err != nil {
			return fmt.Errorf("goja: run program: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// run executes fn synchronously on the event loop, drains all pending
// timers and microtasks, then returns.
func (r *Runtime) run(fn func(rt *gojalib.Runtime) error) error {
	var runErr error
	r.loop.Run(func(rt *gojalib.Runtime) {
		runErr = fn(rt)
	})
	return runErr
}

// RunRender executes fn on the event loop with direct access to the underlying
// goja runtime, draining pending timers and microtasks before returning.
//
// It is the additive entry point for Go-driven renderers (e.g. the nativersc
// renderer) that walk the React element tree natively rather than delegating to
// react-server-dom-webpack's renderToReadableStream. The existing react renderer
// uses CallRenderFunction/DrainStream and never calls this; its behaviour is
// unchanged.
func (r *Runtime) RunRender(fn func(rt *gojalib.Runtime) error) error {
	return r.run(fn)
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	err := r.run(func(rt *gojalib.Runtime) error {
		v, err := rt.RunString(script)
		if err != nil {
			return err
		}
		result = v.Export()
		return nil
	})
	return result, err
}

// Call implements core.JSRuntime.
func (r *Runtime) Call(fn string, args ...any) (any, error) {
	var result any
	err := r.run(func(rt *gojalib.Runtime) error {
		callable, ok := gojalib.AssertFunction(rt.Get(fn))
		if !ok {
			return fmt.Errorf("goja: %q is not a function", fn)
		}
		gojaArgs := make([]gojalib.Value, len(args))
		for i, a := range args {
			gojaArgs[i] = rt.ToValue(a)
		}
		v, err := callable(gojalib.Undefined(), gojaArgs...)
		if err != nil {
			return err
		}
		result = v.Export()
		return nil
	})
	return result, err
}

// Set implements core.JSRuntime.
func (r *Runtime) Set(name string, value any) error {
	return r.run(func(rt *gojalib.Runtime) error {
		return rt.Set(name, value)
	})
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.loop.Stop()
}

// SetRequestContext injects per-request data as __REQUEST__ in the Runtime.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return r.run(func(rt *gojalib.Runtime) error {
		return rt.Set(globals.RequestContext, rt.ToValue(ctx))
	})
}

// SetDependencyInjection injects async bridge functions as __DEPENDENCY_INJECTION__
// properties. Each function is wrapped in a Go goroutine + Promise pattern.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	return r.run(func(rt *gojalib.Runtime) error {
		// Clear existing keys.
		for _, key := range r.di.Keys() {
			r.di.Delete(key) //nolint:errcheck
		}
		for name, fn := range funcs {
			fn := fn                                                    // capture loop variable
			r.di.Set(name, func(c gojalib.FunctionCall) gojalib.Value { //nolint:errcheck
				args := exportArgs(c.Arguments)
				p, resolve, reject := rt.NewPromise()
				go func() {
					result, err := fn(args)
					r.loop.RunOnLoop(func(rt *gojalib.Runtime) {
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

// ── SSRRuntime / SSRPool adapters ─────────────────────────────────────────────

// gojaStreamHandle wraps a StreamSession so it satisfies core.StreamHandle.
type gojaStreamHandle struct{ sess StreamSession }

func (h *gojaStreamHandle) IsNil() bool { return h == nil }

// gojaSSRPool wraps *vmpool.Pool and implements core.SSRPool.
type gojaSSRPool struct {
	pool   *vmpool.Pool[*Runtime]
	logger core.Logger
}

func (p *gojaSSRPool) Acquire() (core.SSRRuntime, error) { return p.pool.Acquire() }
func (p *gojaSSRPool) Release(rt core.SSRRuntime) {
	if r, ok := rt.(*Runtime); ok {
		p.pool.Release(r)
	}
}

// CallRenderFunction implements core.SSRRuntime. It calls StartRender and wraps
// the result in a gojaStreamHandle.
func (r *Runtime) CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error) {
	sess, err := r.startRender(exportName, propsJSON)
	if err != nil {
		return nil, err
	}
	return &gojaStreamHandle{sess: sess}, nil
}

// DrainStream implements core.SSRRuntime. It type-asserts handle to *gojaStreamHandle
// and drains the underlying stream.
func (r *Runtime) DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error) {
	h, ok := handle.(*gojaStreamHandle)
	if !ok {
		return false, fmt.Errorf("goja: DrainStream: unexpected handle type %T", handle)
	}
	return r.drainSession(h.sess, w)
}

// Ensure *Runtime satisfies core.SSRRuntime at compile time.
var _ core.SSRRuntime = (*Runtime)(nil)

// ── StreamSession ──────────────────────────────────────────────────────────────

// StreamSession holds the JS callables and stream value for a single render.
type StreamSession struct {
	PullStreamFn gojalib.Callable
	DecoderObj   *gojalib.Object
	DecodeFn     gojalib.Callable
	Stream       gojalib.Value
}

// startRender looks up __render__ and __pullStream__, instantiates a TextDecoder,
// and calls __render__ to obtain the RSC Flight ReadableStream.
// (Renamed from StartRender; use CallRenderFunction for the core.SSRRuntime interface.)
func (r *Runtime) startRender(exportName, propsJSON string) (StreamSession, error) {
	var sess StreamSession
	err := r.run(func(rt *gojalib.Runtime) error {
		renderFn, ok := gojalib.AssertFunction(rt.Get(globals.RenderFn))
		if !ok {
			return fmt.Errorf("goja: %s is not a function", globals.RenderFn)
		}
		sess.PullStreamFn, ok = gojalib.AssertFunction(rt.Get(globals.PullStreamFn))
		if !ok {
			return fmt.Errorf("goja: %s is not a function", globals.PullStreamFn)
		}

		decoderVal, err := rt.RunString("new TextDecoder()")
		if err != nil {
			return fmt.Errorf("goja: TextDecoder instantiation failed: %w", err)
		}
		sess.DecoderObj = decoderVal.ToObject(rt)
		sess.DecodeFn, ok = gojalib.AssertFunction(sess.DecoderObj.Get("decode"))
		if !ok {
			return fmt.Errorf("goja: TextDecoder.decode is not a function")
		}

		sess.Stream, err = renderFn(gojalib.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		return err
	})
	return sess, err
}

// drainSession polls sess until the stream is done, writing decoded chunks to w.
// Returns (true, nil) if any bytes were written.
func (r *Runtime) drainSession(sess StreamSession, w core.StreamWriter) (bool, error) {
	var wroteAny bool
	for {
		var done, noChunks bool
		if err := r.run(func(rt *gojalib.Runtime) error {
			res, err := sess.PullStreamFn(gojalib.Undefined(), sess.Stream)
			if err != nil {
				return err
			}
			rObj := res.ToObject(rt)
			chunksArr := rObj.Get("chunks").ToObject(rt)
			chunksLen := int(chunksArr.Get("length").ToInteger())
			if chunksLen > 0 {
				var sb strings.Builder
				for i := range chunksLen {
					decoded, err := sess.DecodeFn(sess.DecoderObj, chunksArr.Get(strconv.Itoa(i)))
					if err != nil {
						return err
					}
					sb.WriteString(decoded.String())
				}
				w.WriteRaw([]byte(sb.String())) //nolint:errcheck
				w.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			done = rObj.Get("done").ToBoolean()
			return nil
		}); err != nil {
			return wroteAny, err
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

// clearState removes per-request state from the runtime (used before pooling).
func (r *Runtime) clearState() error {
	return r.run(func(rt *gojalib.Runtime) error {
		rt.Set(globals.RequestContext, gojalib.Undefined()) //nolint:errcheck
		rt.Set(globals.StreamHandle, gojalib.Undefined())   //nolint:errcheck
		rt.Set(globals.OutputChunk, gojalib.Undefined())    //nolint:errcheck
		for _, key := range r.di.Keys() {
			r.di.Delete(key) //nolint:errcheck
		}
		// Reset bridge object to discard memoization Proxy and stale cache.
		r.di = rt.NewObject()
		rt.Set(globals.BridgeObject, r.di) //nolint:errcheck
		return nil
	})
}

// ── VMPool ────────────────────────────────────────────────────────────────────

// VMPool is a bounded pool of *Runtime values backed by a compiled bundle.
type VMPool struct {
	pool *vmpool.Pool[*Runtime]
}

// NewVMPool compiles bundleSource once and creates a bounded pool of Runtimes.
func NewVMPool(bundleSource string, logger core.Logger) (*VMPool, error) {
	prog, err := gojalib.Compile("bundle.js", bundleSource, false)
	if err != nil {
		return nil, fmt.Errorf("goja: pool compile: %w", err)
	}
	p, err := vmpool.New(
		vmpool.Config{MinSize: 1, MaxSize: 64},
		func() (*Runtime, error) { return newRuntime(prog, logger) },
		func(r *Runtime) { _ = r.clearState() },
	)
	if err != nil {
		return nil, fmt.Errorf("goja: pool init: %w", err)
	}
	return &VMPool{pool: p}, nil
}

// Acquire returns a Runtime from the pool. Blocks if all VMs are in use.
// Returns an error if VM creation fails.
func (p *VMPool) Acquire() (*Runtime, error) {
	return p.pool.Acquire()
}

// Release clears per-request state and returns the Runtime to the pool.
func (p *VMPool) Release(r *Runtime) {
	p.pool.Release(r)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func exportArgs(vals []gojalib.Value) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v.Export()
	}
	return out
}
