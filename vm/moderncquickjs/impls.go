package moderncquickjs

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	mquickjs "modernc.org/quickjs"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/framework/contract"
	"github.com/polagonow/pola/framework/globals"
)

var (
	clearJSITmpl = template.Must(template.New("clearJSI").Parse(
		"Object.keys({{.BridgeObject}}).forEach(function(k) { delete {{.BridgeObject}}[k]; });"))
	clearStateTmpl = template.Must(template.New("clearState").Parse(
		"globalThis.{{.RequestContext}} = undefined; Object.keys({{.BridgeObject}}).forEach(function(k) { delete {{.BridgeObject}}[k]; });"))
)

// ── StreamHandle ──────────────────────────────────────────────────────────────

// StreamHandle implements framework.StreamHandle.
type StreamHandle struct {
	Sess *RenderSession
}

// IsNil reports whether the handle holds a valid render session.
func (h *StreamHandle) IsNil() bool { return h.Sess == nil }

// ── framework.VM implementation on *VM ───────────────────────────────────────

// SetRequestContext implements framework.VM.
func (vm *VM) SetRequestContext(ctx map[string]any) error {
	if ctx == nil {
		ctx = map[string]any{}
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("moderncquickjs: marshal request context: %w", err)
	}
	var runErr error
	vm.run(func() {
		_, runErr = vm.inner.Eval(
			"globalThis."+globals.RequestContext+" = ("+string(b)+");",
			mquickjs.EvalGlobal,
		)
	})
	if runErr != nil {
		return fmt.Errorf("moderncquickjs: set request context: %w", runErr)
	}
	return nil
}

// SetBridgeFunctions implements framework.VM.
// Stores the per-request bridge table and installs a synchronous JS wrapper
// on __JSI__ for each function. Wrappers dispatch through __mqjs_call__.
func (vm *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error {
	var runErr error
	vm.run(func() {
		vm.perReqFuncs = funcs

		// Clear existing __JSI__ properties first.
		var clearJSIBuf strings.Builder
		_ = clearJSITmpl.Execute(&clearJSIBuf, struct{ BridgeObject string }{globals.BridgeObject})
		if _, err := vm.inner.Eval(clearJSIBuf.String(), mquickjs.EvalGlobal); err != nil {
			runErr = fmt.Errorf("moderncquickjs: clear __JSI__: %w", err)
			return
		}

		// Install a wrapper for each function.
		for name := range funcs {
			nameJSON, _ := json.Marshal(name)
			var scriptBuf strings.Builder
			_ = jsiWrapperJSTmpl.Execute(&scriptBuf, struct {
				BridgeObject, NameJSON, MQJSCallFn, MQJSErrorKey string
			}{globals.BridgeObject, string(nameJSON), globals.MQJSCallFn, globals.MQJSErrorKey})
			if _, err := vm.inner.Eval(scriptBuf.String(), mquickjs.EvalGlobal); err != nil {
				runErr = fmt.Errorf("moderncquickjs: set bridge function %q: %w", name, err)
				return
			}
		}
	})
	return runErr
}

// CallRenderFunction implements framework.VM.
func (vm *VM) CallRenderFunction(exportName, propsJSON string) (framework.StreamHandle, error) {
	sess := startRender(vm, exportName, propsJSON)
	return &StreamHandle{Sess: sess}, nil
}

// DrainStream implements the streamDrainable interface used by RSCFlightProtocol.
func (vm *VM) DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error) {
	h, ok := handle.(*StreamHandle)
	if !ok {
		return false, fmt.Errorf("moderncquickjs DrainStream: expected *StreamHandle, got %T", handle)
	}
	return DrainStream(vm, w, h.Sess)
}

// ClearState implements framework.VM.
func (vm *VM) ClearState() error {
	var runErr error
	vm.run(func() {
		vm.perReqFuncs = nil
		vm.currentWriter = nil
		vm.wroteAny = false
		var clearStateBuf strings.Builder
		_ = clearStateTmpl.Execute(&clearStateBuf, struct{ RequestContext, BridgeObject string }{globals.RequestContext, globals.BridgeObject})
		_, runErr = vm.inner.Eval(clearStateBuf.String(), mquickjs.EvalGlobal)
	})
	if runErr != nil {
		return fmt.Errorf("moderncquickjs: clear state: %w", runErr)
	}
	return nil
}

// ── VMPool ──────────────────────────────────────────────────────

// VMPool wraps the internal pool and implements framework.VMPool.
type VMPool struct {
	inner *vmPool
}

// NewVMPool creates a framework.VMPool backed by a pre-warmed pool.
func NewVMPool(serverBundle string, bridge contract.BridgeConfig) (*VMPool, error) {
	return &VMPool{inner: newVMPool(serverBundle, bridge)}, nil
}

// Acquire implements framework.VMPool.
func (p *VMPool) Acquire() framework.VM { return p.inner.Acquire() }

// Release implements framework.VMPool.
func (p *VMPool) Release(vm framework.VM) { p.inner.Release(vm.(*VM)) }

// ── VMFactory ───────────────────────────────────────────────────

// VMFactory implements framework.VMFactory for modernc.org/quickjs.
type VMFactory struct {
	src string
}

// NewVMFactory stores serverBundle and returns a factory.
func NewVMFactory(serverBundle []byte) (*VMFactory, error) {
	return &VMFactory{src: string(serverBundle)}, nil
}

// New implements framework.VMFactory.
func (f *VMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
	return newVM(f.src, bridge)
}
