// Package quickjsgo provides a QuickJS-backed VM implementation for the Pola
// framework using github.com/buke/quickjs-go.
//
// Async bridge functions resolve via ctx.Schedule, and the entire render is
// driven by ctx.Await on a JS async function. This handles both native Promise
// microtasks and Go-scheduled callbacks in a single cooperative polling loop.
//
// Note: unlike the streaming Goja/Sobek/v8go implementations, quickjsgo
// collects all rendered output in JS before writing to the StreamWriter.
package quickjsgo

import (
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/polagonow/pola/framework/contract"
	"github.com/polagonow/pola/framework/globals"
	"github.com/polagonow/pola/vm/quickjsgo/polyfill"

	quickjs "github.com/buke/quickjs-go"
)

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
		// Yield via a native Promise so ctx.Await() can run pending JS
		// microtasks and Go-scheduled bridge callbacks before we poll again.
		await Promise.resolve();
	}
};
`))
	consoleJSTmpl = template.Must(template.New("console").Parse(`
globalThis.console = {
	log:   function() { {{.QJSLogFn}}("LOG",  Array.prototype.slice.call(arguments)); },
	warn:  function() { {{.QJSLogFn}}("WARN", Array.prototype.slice.call(arguments)); },
	error: function() { {{.QJSLogFn}}("ERR",  Array.prototype.slice.call(arguments)); },
	info:  function() { {{.QJSLogFn}}("INFO", Array.prototype.slice.call(arguments)); },
};
`))
	renderAsyncJS string
)

func init() {
	var b strings.Builder
	if err := renderAsyncJSTmpl.Execute(&b, struct {
		RenderAsyncFn, RenderFn, DrainMicrotasksFn, PullStreamFn, OutputChunk string
	}{globals.RenderAsyncFn, globals.RenderFn, globals.DrainMicrotasksFn, globals.PullStreamFn, globals.OutputChunk}); err != nil {
		panic("quickjsgo: renderAsync template: " + err.Error())
	}
	renderAsyncJS = b.String()
}

// VM holds a QuickJS runtime + context. All JS operations are serialised via mu.
// Bridge goroutines enqueue callbacks through ctx.Schedule (thread-safe).
type VM struct {
	rt     *quickjs.Runtime
	ctx    *quickjs.Context
	mu     sync.Mutex
	jsi    *quickjs.Value // persistent __JSI__ object; freed in close()
	bridge contract.BridgeConfig
}

// run executes fn under the VM mutex.
func (vm *VM) run(fn func()) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	fn()
}

// VMPool manages a pool of pre-warmed VMs.
type VMPool struct {
	pool   sync.Pool
	bridge contract.BridgeConfig
	src    string
}

// NewVMPool compiles (well, stores) the server bundle and returns a pool.
// One VM is eagerly created to surface startup errors immediately.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	p := &VMPool{bridge: bridge, src: serverBundle}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(serverBundle, bridge)
			if err != nil {
				panic(fmt.Sprintf("quickjsgo: pool create: %v", err))
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

// newVM creates a runtime + context, installs globals/polyfills/bundle.
func newVM(src string, bridge contract.BridgeConfig) (*VM, error) {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()

	vm := &VM{rt: rt, ctx: ctx, bridge: bridge}
	var initErr error

	vm.run(func() {
		if err := evalOrErr(ctx, `
globalThis.global = globalThis;
globalThis.globalThis = globalThis;
`, "globals.js"); err != nil {
			initErr = err
			return
		}

		// Console — install a Go-backed __qjs_log__, wire console in JS.
		logFn := ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qCtx.NewUndefined()
			}
			level := args[0].String()
			arr := args[1]
			lenVal := arr.Get("length")
			n := int(lenVal.ToInt32())
			lenVal.Free()
			parts := make([]string, n)
			for i := 0; i < n; i++ {
				elem := arr.GetIdx(int64(i))
				parts[i] = elem.String()
				elem.Free()
			}
			fmt.Printf("[VM:%s] %v\n", level, parts)
			return qCtx.NewUndefined()
		})
		setGlobal(ctx, globals.QJSLogFn, logFn)
		var consoleJS strings.Builder
		if err := consoleJSTmpl.Execute(&consoleJS, struct{ QJSLogFn string }{globals.QJSLogFn}); err != nil {
			initErr = fmt.Errorf("quickjsgo: console template: %w", err)
			return
		}
		if err := evalOrErr(ctx, consoleJS.String(), "console.js"); err != nil {
			initErr = err
			return
		}

		if err := evalOrErr(ctx, `
globalThis.process = { env: { NODE_ENV: "production" } };
globalThis.performance = { now: function() { return 0; } };
`, "proc.js"); err != nil {
			initErr = err
			return
		}

		if err := evalOrErr(ctx, "globalThis."+globals.BridgeObject+" = {};", "jsi.js"); err != nil {
			initErr = err
			return
		}

		// Keep a Go reference to __JSI__ for SetBridgeFunctions.
		jsiVal := ctx.Eval(globals.BridgeObject, quickjs.EvalFileName("get_jsi.js"))
		if jsiVal.IsException() {
			initErr = fmt.Errorf("quickjsgo: get __JSI__: %w", ctx.Exception())
			jsiVal.Free()
			return
		}
		vm.jsi = jsiVal

		// Global bridge functions (synchronous).
		for name, fn := range bridge.Globals {
			bridgeFn := ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
				goArgs := exportArgs(args)
				result, err := fn(goArgs)
				if err != nil {
					qCtx.ThrowError(err) //nolint:errcheck
					return qCtx.NewUndefined()
				}
				val, marshalErr := qCtx.Marshal(result)
				if marshalErr != nil {
					qCtx.ThrowError(marshalErr) //nolint:errcheck
					return qCtx.NewUndefined()
				}
				return val
			})
			setGlobal(ctx, name, bridgeFn)
		}

		// Polyfills (microtask, TextEncoder, ReadableStream, etc.).
		if err := polyfill.Enable(ctx); err != nil {
			initErr = fmt.Errorf("quickjsgo: polyfills: %w", err)
			return
		}

		// Run the server bundle.
		if err := evalOrErr(ctx, src, "bundle.js"); err != nil {
			initErr = fmt.Errorf("quickjsgo: run bundle: %w", err)
			return
		}

		// Install the async render collector (depends on bundle's __render__).
		if err := evalOrErr(ctx, renderAsyncJS, "render_async.js"); err != nil {
			initErr = fmt.Errorf("quickjsgo: render async: %w", err)
		}
	})

	if initErr != nil {
		ctx.Close()
		rt.Close()
		return nil, initErr
	}
	return vm, nil
}

// evalOrErr runs src in ctx, frees the result, and returns any JS exception.
func evalOrErr(ctx *quickjs.Context, src, name string) error {
	ret := ctx.Eval(src, quickjs.EvalFileName(name))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("%s: %w", name, ctx.Exception())
	}
	return nil
}

// setGlobal sets name on ctx.Globals().
// Set transfers ownership of val; Globals() returns a cached singleton — neither is freed here.
func setGlobal(ctx *quickjs.Context, name string, val *quickjs.Value) {
	g := ctx.Globals()
	g.Set(name, val) //nolint:errcheck
}

// exportArgs converts []*quickjs.Value to []interface{} using String() for
// simple values. Complex objects are marshalled to JSON then exported.
func exportArgs(args []*quickjs.Value) []interface{} {
	out := make([]interface{}, len(args))
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
