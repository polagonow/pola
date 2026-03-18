// package qjs provides a QuickJS-backed VM implementation for the
// GoJSX framework using github.com/fastschema/qjs (CGo-free, Wazero-based).
//
// Async bridge functions are created with ctx.Function(fn, true) (isAsync=true).
// The goroutine spawned by each bridge call resolves/rejects the promise via
// this.Promise().Resolve / this.Promise().Reject. The render is driven by
// promise.Await() on the __renderAsync__ promise.
//
// Note: like the buke/quickjs-go implementation, all rendered output is
// collected via the __outputChunk__ Go function called synchronously from JS
// during promise.Await(), enabling React Suspense streaming.
package qjs

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"gojsx/framework/contract"
	"gojsx/framework/globals"
	"gojsx/vm/qjs/polyfill"

	qjs "github.com/fastschema/qjs"
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
		// Yield via a native Promise so the event loop can run pending
		// bridge-function callbacks before we poll again.
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
		panic("qjs: renderAsync template: " + err.Error())
	}
	renderAsyncJS = b.String()
}

// VM holds a qjs runtime + context. All JS operations are serialised
// via mu except during promise.Await(), which releases the lock so bridge
// goroutines can call back into the context.
type VM struct {
	rt     *qjs.Runtime
	ctx    *qjs.Context
	mu     sync.Mutex
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

// NewVMPool compiles (stores) the server bundle and returns a pool.
// One VM is eagerly created to surface startup errors immediately.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	p := &VMPool{bridge: bridge, src: serverBundle}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newVM(serverBundle, bridge)
			if err != nil {
				panic(fmt.Sprintf("qjs: pool create: %v", err))
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
	rt, err := qjs.New()
	if err != nil {
		return nil, fmt.Errorf("qjs: new runtime: %w", err)
	}
	ctx := rt.Context()

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
		ctx.SetFunc(globals.QJSLogFn, func(this *qjs.This) (*qjs.Value, error) {
			args := this.Args()
			if len(args) < 2 {
				return this.Context().NewUndefined(), nil
			}
			level := args[0].String()
			arr := args[1]
			lenVal := arr.GetPropertyStr("length")
			n := int(lenVal.Int32())
			lenVal.Free()
			parts := make([]string, n)
			for i := 0; i < n; i++ {
				elem := arr.GetPropertyIndex(int64(i))
				parts[i] = elem.String()
				elem.Free()
			}
			fmt.Printf("[VM:%s] %v\n", level, parts)
			return this.Context().NewUndefined(), nil
		})

		var consoleJS strings.Builder
		if err := consoleJSTmpl.Execute(&consoleJS, struct{ QJSLogFn string }{globals.QJSLogFn}); err != nil {
			initErr = fmt.Errorf("qjs: console template: %w", err)
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

		// Global bridge functions (synchronous).
		for name, fn := range bridge.Globals {
			ctx.SetFunc(name, func(this *qjs.This) (*qjs.Value, error) {
				goArgs := exportArgs(this.Args())
				result, err := fn(goArgs)
				if err != nil {
					return nil, err
				}
				b, marshalErr := json.Marshal(result)
				if marshalErr != nil {
					return nil, marshalErr
				}
				return this.Context().ParseJSON(string(b)), nil
			})
		}

		// Polyfills (microtask, TextEncoder, ReadableStream, etc.).
		if err := polyfill.Enable(ctx); err != nil {
			initErr = fmt.Errorf("qjs: polyfills: %w", err)
			return
		}

		// Run the server bundle.
		if err := evalOrErr(ctx, src, "bundle.js"); err != nil {
			initErr = fmt.Errorf("qjs: run bundle: %w", err)
			return
		}

		// Install the async render collector (depends on bundle's __render__).
		if err := evalOrErr(ctx, renderAsyncJS, "render_async.js"); err != nil {
			initErr = fmt.Errorf("qjs: render async: %w", err)
		}
	})

	if initErr != nil {
		rt.Close()
		return nil, initErr
	}
	return vm, nil
}

// evalOrErr evaluates src in ctx, frees the result value, and returns any error.
func evalOrErr(ctx *qjs.Context, src, name string) error {
	val, err := ctx.Eval(name, qjs.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

// exportArgs converts []*qjs.Value to []interface{}.
func exportArgs(args []*qjs.Value) []interface{} {
	out := make([]interface{}, len(args))
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
