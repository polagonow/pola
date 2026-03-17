package v8go

import (
	"fmt"
	"sync"

	"gojsx/framework/contract"
	"gojsx/vm/eventloop"
	"gojsx/vm/v8go/polyfill"
	"gojsx/vm/v8go/polyfill/console"
	"gojsx/vm/v8go/polyfill/globals"
	"gojsx/vm/v8go/polyfill/jsi"

	v8 "rogchap.com/v8go"
)

// V8VM wraps a V8 isolate + context with a goroutine-pinned event loop.
// All V8 operations run on the loop goroutine via RunSync / RunAsync.
type V8VM struct {
	iso     *v8.Isolate
	ctx     *v8.Context
	loop    *eventloop.EventLoop
	bridge  contract.BridgeConfig
	jsiKeys []string // keys currently set on __JSI__; cleared in ClearState
}

// V8VMPool manages a pool of pre-warmed V8VMs.
type V8VMPool struct {
	pool   sync.Pool
	bridge contract.BridgeConfig
	source string
}

// NewV8VMPool creates a pool backed by the given server-bundle source.
// One VM is eagerly created to surface startup errors immediately.
func NewV8VMPool(serverBundle string, bridge contract.BridgeConfig) (*V8VMPool, error) {
	p := &V8VMPool{bridge: bridge, source: serverBundle}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := newV8VM(serverBundle, bridge)
			if err != nil {
				panic(fmt.Sprintf("v8: pool create: %v", err))
			}
			return vm
		},
	}
	// Eagerly create one VM to catch startup errors.
	vm := p.pool.New()
	p.pool.Put(vm)
	return p, nil
}

// Acquire returns a V8VM from the pool.
func (p *V8VMPool) Acquire() *V8VM {
	return p.pool.Get().(*V8VM)
}

// Release clears per-request state and returns the VM to the pool.
func (p *V8VMPool) Release(vm *V8VM) {
	_ = vm.ClearState()
	p.pool.Put(vm)
}

// newV8VM creates a fresh isolate, installs globals + polyfills, and runs the bundle.
func newV8VM(source string, bridge contract.BridgeConfig) (*V8VM, error) {
	iso := v8.NewIsolate()
	loop := eventloop.New(true)
	vm := &V8VM{iso: iso, loop: loop, bridge: bridge}

	var initErr error
	loop.RunSync(func() {
		ctx := v8.NewContext(iso)
		vm.ctx = ctx
		loop.SetCheckpoint(ctx.PerformMicrotaskCheckpoint)
		global := ctx.Global()

		// Self-referential globals.
		global.Set("global", global)     //nolint:errcheck
		global.Set("globalThis", global) //nolint:errcheck

		if err := console.Enable(iso, ctx); err != nil {
			initErr = fmt.Errorf("v8: %w", err)
			return
		}
		if err := globals.Enable(ctx); err != nil {
			initErr = fmt.Errorf("v8: %w", err)
			return
		}
		if err := jsi.Enable(ctx); err != nil {
			initErr = fmt.Errorf("v8: %w", err)
			return
		}

		// Global bridge functions (synchronous).
		for name, fn := range bridge.Globals {
			name, fn := name, fn
			ft := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
				args := exportV8Args(info.Args())
				result, err := fn(args)
				if err != nil {
					errVal, _ := v8.NewValue(iso, err.Error())
					iso.ThrowException(errVal) //nolint:errcheck
					return nil
				}
				val, convErr := goToV8Value(iso, ctx, result)
				if convErr != nil {
					errVal, _ := v8.NewValue(iso, convErr.Error())
					iso.ThrowException(errVal) //nolint:errcheck
					return nil
				}
				return val
			})
			global.Set(name, ft.GetFunction(ctx)) //nolint:errcheck
		}

		// Polyfills (microtask, TextEncoder, ReadableStream, etc.).
		if err := polyfill.Enable(ctx); err != nil {
			initErr = fmt.Errorf("v8: polyfills: %w", err)
			return
		}

		// Run the compiled server bundle.
		if _, err := ctx.RunScript(source, "bundle.js"); err != nil {
			initErr = fmt.Errorf("v8: run bundle: %w", err)
			return
		}
	})

	if initErr != nil {
		loop.Stop()
		iso.Dispose()
		return nil, initErr
	}
	return vm, nil
}

// run executes fn synchronously on the event loop and returns any error.
func (vm *V8VM) run(fn func() error) error {
	var runErr error
	vm.loop.RunSync(func() {
		runErr = fn()
	})
	return runErr
}
