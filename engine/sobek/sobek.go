// Package sobek provides a Sobek-backed JSEngine implementation for the Pola
// framework. Sobek (github.com/grafana/sobek) is a fork of Goja with the same
// API surface. This implementation uses a channel-based event loop to serialise
// all runtime access onto a single goroutine. Bridge async functions resolve
// via RunAsync so their Promise continuations are drained on the next checkpoint.
//
// Build tag: sobek
package sobek

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/engine/eventloop"
	"github.com/polagonow/pola/engine/polyfill"

	sobeklib "github.com/grafana/sobek"
)

// drainProg is a pre-compiled no-op that triggers Sobek's internal Promise
// microtask queue to drain. Called as a checkpoint after every loop task.
var drainProg, _ = sobeklib.Compile("drain.js", "0", false)

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine is a Sobek-backed JSEngine.
type Engine struct {
	prog   *sobeklib.Program
	logger core.Logger
}

// SetLogger implements core.LogAware.
func (e *Engine) SetLogger(l core.Logger) { e.logger = l }

// New compiles bundleSource and returns a ready Engine.
func New(bundleSource string) (*Engine, error) {
	prog, err := sobeklib.Compile("bundle.js", bundleSource, false)
	if err != nil {
		return nil, fmt.Errorf("sobek: compile: %w", err)
	}
	return &Engine{prog: prog}, nil
}

// Name implements core.JSEngine.
func (e *Engine) Name() string { return "sobek" }

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
	return newRuntime(e.prog, e.logger)
}

// ── Runtime ───────────────────────────────────────────────────────────────────

// Runtime wraps a Sobek runtime with a goroutine-serialised event loop.
type Runtime struct {
	rt   *sobeklib.Runtime
	loop *eventloop.EventLoop
	di   *sobeklib.Object
}

func newRuntime(prog *sobeklib.Program, logger core.Logger) (*Runtime, error) {
	loop := eventloop.New(false)
	r := &Runtime{loop: loop}

	var initErr error
	loop.RunSync(func() {
		rt := sobeklib.New()
		r.rt = rt
		loop.SetCheckpoint(func() { rt.RunProgram(drainProg) }) //nolint:errcheck

		rt.Set("globalThis", rt.GlobalObject()) //nolint:errcheck

		r.di = rt.NewObject()
		rt.Set(globals.BridgeObject, r.di) //nolint:errcheck

		// Install __pola_log__ so ConsoleBridge polyfill can wire console.* to the logger.
		rt.Set(globals.PolaLogFn, func(c sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
			if len(c.Arguments) >= 2 {
				polyfill.LogAtLevel(logger, "sobek", c.Arguments[0].String(), c.Arguments[1].String())
			}
			return sobeklib.Undefined()
		})

		// Install polyfills from the default registry.
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
				initErr = fmt.Errorf("sobek: polyfill %s: %w", src.ID, err)
				return
			}
		}

		if _, err := rt.RunProgram(prog); err != nil {
			initErr = fmt.Errorf("sobek: run bundle: %w", err)
		}
	})

	if initErr != nil {
		loop.Stop()
		return nil, initErr
	}
	return r, nil
}

func (r *Runtime) run(fn func(rt *sobeklib.Runtime) error) error {
	var runErr error
	r.loop.RunSync(func() {
		runErr = fn(r.rt)
	})
	return runErr
}

// Eval implements core.JSRuntime.
func (r *Runtime) Eval(script string) (any, error) {
	var result any
	err := r.run(func(rt *sobeklib.Runtime) error {
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
	err := r.run(func(rt *sobeklib.Runtime) error {
		callable, ok := sobeklib.AssertFunction(rt.Get(fn))
		if !ok {
			return fmt.Errorf("sobek: %q is not a function", fn)
		}
		vals := make([]sobeklib.Value, len(args))
		for i, a := range args {
			vals[i] = rt.ToValue(a)
		}
		v, err := callable(sobeklib.Undefined(), vals...)
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
	return r.run(func(rt *sobeklib.Runtime) error {
		return rt.Set(name, value)
	})
}

// Dispose implements core.JSRuntime.
func (r *Runtime) Dispose() {
	r.loop.Stop()
}

// SetRequestContext injects per-request data as __REQUEST__.
func (r *Runtime) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	return r.run(func(rt *sobeklib.Runtime) error {
		return rt.Set(globals.RequestContext, rt.ToValue(ctx))
	})
}

// SetDependencyInjection installs async bridge functions on __DEPENDENCY_INJECTION__.
func (r *Runtime) SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error {
	return r.run(func(rt *sobeklib.Runtime) error {
		for _, key := range r.di.Keys() {
			r.di.Delete(key) //nolint:errcheck
		}
		for name, fn := range funcs {
			fn := fn                                                      // capture
			r.di.Set(name, func(c sobeklib.FunctionCall) sobeklib.Value { //nolint:errcheck
				args := exportArgs(c.Arguments)
				p, resolve, reject := rt.NewPromise()
				go func() {
					result, err := fn(args)
					r.loop.RunAsync(func() {
						if err != nil {
							reject(rt.ToValue(err.Error())) //nolint:errcheck
						} else {
							resolve(rt.ToValue(result)) //nolint:errcheck
						}
					})
				}()
				return rt.ToValue(p)
			})
		}
		return nil
	})
}

// StreamSession holds the JS callables for a single render.
type StreamSession struct {
	PullStreamFn sobeklib.Callable
	DecoderObj   *sobeklib.Object
	DecodeFn     sobeklib.Callable
	Stream       sobeklib.Value
}

// StartRender obtains the RSC Flight ReadableStream.
func (r *Runtime) StartRender(exportName, propsJSON string) (StreamSession, error) {
	var sess StreamSession
	err := r.run(func(rt *sobeklib.Runtime) error {
		renderFn, ok := sobeklib.AssertFunction(rt.Get(globals.RenderFn))
		if !ok {
			return fmt.Errorf("sobek: %s is not a function", globals.RenderFn)
		}
		sess.PullStreamFn, ok = sobeklib.AssertFunction(rt.Get(globals.PullStreamFn))
		if !ok {
			return fmt.Errorf("sobek: %s is not a function", globals.PullStreamFn)
		}

		decoderVal, err := rt.RunString("new TextDecoder()")
		if err != nil {
			return fmt.Errorf("sobek: TextDecoder failed: %w", err)
		}
		sess.DecoderObj = decoderVal.ToObject(rt)
		sess.DecodeFn, ok = sobeklib.AssertFunction(sess.DecoderObj.Get("decode"))
		if !ok {
			return fmt.Errorf("sobek: TextDecoder.decode is not a function")
		}

		sess.Stream, err = renderFn(sobeklib.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		return err
	})
	return sess, err
}

// DrainStream polls sess until done, writing decoded chunks to w.
func (r *Runtime) drainSession(sess StreamSession, w core.StreamWriter) (bool, error) {
	var wroteAny bool
	for {
		var done, noChunks bool
		if err := r.run(func(rt *sobeklib.Runtime) error {
			res, err := sess.PullStreamFn(sobeklib.Undefined(), sess.Stream)
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

func (r *Runtime) clearState() error {
	return r.run(func(rt *sobeklib.Runtime) error {
		rt.Set(globals.RequestContext, sobeklib.Undefined()) //nolint:errcheck
		rt.Set(globals.StreamHandle, sobeklib.Undefined())   //nolint:errcheck
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

// VMPool manages a pool of pre-warmed Sobek Runtimes.
type VMPool struct {
	pool   sync.Pool
	prog   *sobeklib.Program
	logger core.Logger
}

// NewVMPool compiles the server bundle once and creates a pool.
func NewVMPool(serverBundle string, logger core.Logger) (*VMPool, error) {
	prog, err := sobeklib.Compile("bundle.js", serverBundle, false)
	if err != nil {
		return nil, fmt.Errorf("sobek: pool compile: %w", err)
	}
	p := &VMPool{prog: prog, logger: logger}
	p.pool = sync.Pool{
		New: func() any {
			r, err := newRuntime(prog, logger)
			if err != nil {
				panic(fmt.Sprintf("sobek: pool create: %v", err))
			}
			return r
		},
	}
	// Eagerly create one Runtime to catch startup errors immediately.
	r, err := newRuntime(prog, logger)
	if err != nil {
		return nil, fmt.Errorf("sobek: pool create: %w", err)
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
	_ = r.clearState()
	p.pool.Put(r)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func exportArgs(vals []sobeklib.Value) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v.Export()
	}
	return out
}
