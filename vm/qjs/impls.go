package qjs

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"gojsx/framework"
	"gojsx/framework/contract"
	"gojsx/framework/globals"

	qjs "github.com/fastschema/qjs"
)

var (
	clearJSITmpl = template.Must(template.New("clearJSI").Parse(
		"Object.keys({{.BridgeObject}}).forEach(function(k) { delete {{.BridgeObject}}[k]; })"))
	clearStateTmpl = template.Must(template.New("clearState").Parse(
		"{{.RequestContext}} = undefined; {{.StreamHandle}} = undefined; {{.OutputChunk}} = undefined; Object.keys({{.BridgeObject}}).forEach(function(k) { delete {{.BridgeObject}}[k]; });"))
)

// ── FastSchemaQJSStreamHandle ─────────────────────────────────────────────────

// FastSchemaQJSStreamHandle implements framework.StreamHandle.
type FastSchemaQJSStreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *FastSchemaQJSStreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("qjs: marshal request context: %w", err)
	}
	vm.run(func() {
		val := vm.ctx.ParseJSON(string(b))
		vm.ctx.Global().SetPropertyStr(globals.RequestContext, val)
		// val ownership transferred to the global property; do not free.
	})
	return nil
}

// bridgeResult carries the outcome of an async Go bridge call.
type bridgeResult struct {
	val interface{}
	err error
}

// SetBridgeFunctions implements framework.VM.
//
// Bridge functions are intentionally synchronous from the JS/WASM perspective:
// the host function blocks on a buffered channel while the Go goroutine runs,
// then returns the marshalled result back into WASM via a re-entrant call on
// the same goroutine. This ensures WASM is never called concurrently from
// multiple goroutines (Wazero does not support that).
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	vm.run(func() {
		// Clear existing keys.
		var clearJSIBuf strings.Builder
		_ = clearJSITmpl.Execute(&clearJSIBuf, struct{ BridgeObject string }{globals.BridgeObject})
		if _, err := vm.ctx.Eval("clear_jsi.js", qjs.Code(clearJSIBuf.String())); err != nil {
			_ = err // non-fatal
		}

		// Install synchronous bridge functions on __JSI__.
		jsi := vm.ctx.Global().GetPropertyStr(globals.BridgeObject)
		defer jsi.Free()

		for name, fn := range funcs {
			jsFn := vm.ctx.Function(func(this *qjs.This) (*qjs.Value, error) {
				goArgs := exportArgs(this.Args())

				// Buffered so the goroutine never blocks even if we time out.
				ch := make(chan bridgeResult, 1)
				go func() {
					v, e := fn(goArgs)
					ch <- bridgeResult{v, e}
				}()

				// Block the host-function call (suspending the WASM stack) until
				// the goroutine finishes. The goroutine itself never calls WASM,
				// so there is no concurrent WASM access.
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
				// Re-entrant WASM call — safe because it is on the same goroutine.
				return this.Context().ParseJSON(string(b)), nil
			})
			jsi.SetPropertyStr(name, jsFn)
			// jsFn ownership transferred to jsi property; do not free.
		}
	})
	return nil
}

// CallRenderFunction implements framework.VM.
func (vm *VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess, err := StartRender(vm, exportName, propsJSON)
	if err != nil {
		return &FastSchemaQJSStreamHandle{}, err
	}
	return &FastSchemaQJSStreamHandle{Sess: sess}, nil
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	qjsHandle, ok := handle.(*FastSchemaQJSStreamHandle)
	if !ok {
		return false, fmt.Errorf("qjs DrainStream: expected *FastSchemaQJSStreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, qjsHandle.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	vm.run(func() {
		var clearStateBuf strings.Builder
		_ = clearStateTmpl.Execute(&clearStateBuf, struct {
			RequestContext, StreamHandle, OutputChunk, BridgeObject string
		}{globals.RequestContext, globals.StreamHandle, globals.OutputChunk, globals.BridgeObject})
		_, _ = vm.ctx.Eval("clear_state.js", qjs.Code(clearStateBuf.String()))
	})
	return nil
}

// ── FastSchemaQJSVMPool ───────────────────────────────────────────────────────

// FastSchemaQJSVMPool wraps *VMPool and implements framework.VMPool.
type FastSchemaQJSVMPool struct {
	inner *VMPool
}

// NewFastSchemaQJSVMPool creates a framework.VMPool backed by a pre-warmed pool.
func NewFastSchemaQJSVMPool(serverBundle string, bridge contract.BridgeConfig) (*FastSchemaQJSVMPool, error) {
	inner, err := NewVMPool(serverBundle, bridge)
	if err != nil {
		return nil, err
	}
	return &FastSchemaQJSVMPool{inner: inner}, nil
}

// Acquire implements framework.VMPool.
func (p *FastSchemaQJSVMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *FastSchemaQJSVMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── FastSchemaQJSVMFactory ────────────────────────────────────────────────────

// FastSchemaQJSVMFactory implements framework.VMFactory for qjs.
type FastSchemaQJSVMFactory struct {
	src string
}

// NewFastSchemaQJSVMFactory stores serverBundle and returns a factory ready to create VMs.
func NewFastSchemaQJSVMFactory(serverBundle []byte) (*FastSchemaQJSVMFactory, error) {
	return &FastSchemaQJSVMFactory{src: string(serverBundle)}, nil
}

// New implements framework.VMFactory.
func (f *FastSchemaQJSVMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newVM(f.src, bridge)
}
