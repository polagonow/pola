package react

import (
	"context"
	"encoding/json"
	"fmt"

	"gojsx/framework"
	"gojsx/framework/contract"
)

// VMRenderer implements framework.Renderer using the VM pool and stream protocol.
// It acquires a VM, sets up the request context and bridge, calls the render
// function, drains the stream, and releases the VM.
type VMRenderer struct {
	pool         framework.VMPool
	protocol     framework.StreamProtocol
	globalBridge contract.BridgeConfig
}

// NewVMRenderer creates a Renderer backed by pool and protocol.
func NewVMRenderer(
	pool framework.VMPool, protocol framework.StreamProtocol, globalBridge contract.BridgeConfig,
) *VMRenderer {
	return &VMRenderer{pool: pool, protocol: protocol, globalBridge: globalBridge}
}

// Render implements framework.Renderer.
func (r *VMRenderer) Render(_ context.Context, w framework.StreamWriter, opts contract.RenderOpts) error {
	vm := r.pool.Acquire()
	defer r.pool.Release(vm)

	if err := vm.SetRequestContext(opts.RequestContext); err != nil {
		return fmt.Errorf("renderer: set context: %w", err)
	}

	jsi := r.globalBridge.Context
	if opts.Bridge != nil {
		jsi = opts.Bridge.Context
	}
	if err := vm.SetBridgeFunctions(jsi); err != nil {
		return fmt.Errorf("renderer: set bridge: %w", err)
	}

	propsJSON, err := json.Marshal(opts.Props)
	if err != nil {
		return fmt.Errorf("renderer: marshal props: %w", err)
	}

	handle, err := vm.CallRenderFunction(opts.ExportName, string(propsJSON))
	if err != nil {
		return fmt.Errorf("renderer: call render: %w", err)
	}

	return r.protocol.Drain(vm, handle, w)
}
