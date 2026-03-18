package react

import (
	"encoding/json"
	"fmt"

	"github.com/polagonow/pola/framework/contract"
	vmgoja "github.com/polagonow/pola/vm/goja"
)

// RenderOptions controls a single page render.
type RenderOptions = contract.RenderOpts

// Renderer drives server renders using the Goja VM pool.
type Renderer struct {
	pool     *vmgoja.VMPool
	manifest ClientManifest
}

// NewRenderer creates a Renderer backed by the given VM pool.
func NewRenderer(pool *vmgoja.VMPool, manifest ClientManifest) *Renderer {
	return &Renderer{pool: pool, manifest: manifest}
}

// Render calls __render__ in the VM, collects the RSC Flight ReadableStream via
// a Go for loop, and writes all chunks to fw.
//
// Each iteration calls vm.run (loop.Run), which drains all pending timers and
// RunOnLoop callbacks before returning. This lets goroutine-driven async/await
// in server components resolve between iterations without any setTimeout scheduling.
func (r *Renderer) Render(fw *FlightWriter, vm *vmgoja.VM, opts RenderOptions) error {
	if err := vm.SetRequestContext(opts.RequestContext); err != nil {
		return fmt.Errorf("render: set context: %w", err)
	}
	jsi := r.pool.Bridge().Context
	if opts.Bridge != nil {
		jsi = opts.Bridge.Context
	}
	if err := vm.SetJSI(jsi); err != nil {
		return fmt.Errorf("render: set jsi: %w", err)
	}
	propsJSON, err := json.Marshal(opts.Props)
	if err != nil {
		return fmt.Errorf("render: marshal props: %w", err)
	}

	sess, err := vmgoja.StartRender(vm, opts.ExportName, string(propsJSON))
	if err != nil {
		return fmt.Errorf("render: setup: %w", err)
	}

	wroteAny, err := vmgoja.DrainStream(vm, fw, sess)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if !wroteAny {
		return fmt.Errorf("render: no Flight output written")
	}
	return nil
}

// LoadManifest parses the client component manifest JSON.
func LoadManifest(data []byte) (ClientManifest, error) {
	var m ClientManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return m, nil
}
