package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// RenderOptions controls a single page render.
type RenderOptions struct {
	ExportName     string
	Props          map[string]any
	RequestContext map[string]any
}

// Renderer drives server renders using the Goja VM pool.
type Renderer struct {
	pool     *VMPool
	manifest ClientManifest
}

// NewRenderer creates a Renderer backed by the given VM pool.
func NewRenderer(pool *VMPool, manifest ClientManifest) *Renderer {
	return &Renderer{pool: pool, manifest: manifest}
}

// Render calls __render__ in the VM, collects the RSC Flight ReadableStream via
// a setTimeout-based poll loop, and writes all chunks to fw.
//
// The poll loop runs inside vm.run() (which wraps loop.Run). Because loop.Run
// keeps processing the event loop until it is fully drained, each
// setTimeout(__poll__, 0) yield lets native Promises from async server
// components resolve before the next poll. This enables true async/await
// inside server components without any manual microtask flushing.
func (r *Renderer) Render(fw *FlightWriter, vm *VM, opts RenderOptions) error {
	if err := vm.SetRequestContext(opts.RequestContext); err != nil {
		return fmt.Errorf("render: set context: %w", err)
	}

	propsJSON, err := json.Marshal(opts.Props)
	if err != nil {
		return fmt.Errorf("render: marshal props: %w", err)
	}

	// Safety: escape export name to avoid injection
	exportName := strings.ReplaceAll(opts.ExportName, `"`, `\"`)

	var flight string
	var renderErr error

	err = vm.run(func(rt *goja.Runtime) error {
		// __gojsx_capture__ is called from JS when the stream is fully drained.
		rt.Set("__gojsx_capture__", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				flight = call.Arguments[0].String()
			}
			return goja.Undefined()
		})

		// Start the render and schedule the first poll tick.
		// Each __poll__ call drains one batch of stream chunks; if the stream
		// is not done yet it reschedules itself via setTimeout(0), yielding
		// control back to the goja_nodejs event loop so that native Promises
		// (async/await in server components) can resolve between ticks.
		script := fmt.Sprintf(`(function() {
	__gojsx_stream__ = __render__(%q, %q);
	var dec = new TextDecoder();
	var out = "";
	var safety = 0;
	function __poll__() {
		if (safety++ > 2000) { __gojsx_capture__(out); return; }
		var r = __pullStream__(__gojsx_stream__);
		for (var i = 0; i < r.chunks.length; i++) {
			out += dec.decode(r.chunks[i]);
		}
		if (r.done) {
			__gojsx_capture__(out);
		} else {
			setTimeout(__poll__, 0);
		}
	}
	setTimeout(__poll__, 0);
})();`, exportName, string(propsJSON))

		if _, err := rt.RunScript("render.js", script); err != nil {
			renderErr = err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("render: event loop: %w", err)
	}
	if renderErr != nil {
		return fmt.Errorf("render: __render__: %w", renderErr)
	}
	if flight == "" {
		return fmt.Errorf("render: empty Flight output")
	}

	if _, err := fw.WriteRaw([]byte(flight)); err != nil {
		return fmt.Errorf("render: write: %w", err)
	}
	fw.Flush()

	// Clean up per-request globals for next request.
	_ = vm.run(func(rt *goja.Runtime) error {
		rt.Set("__gojsx_stream__", goja.Undefined())
		rt.Set("__gojsx_capture__", goja.Undefined())
		return nil
	})

	return nil
}

// ClientManifest maps moduleId → ClientRef.
type ClientManifest map[string]ClientRef

// LoadManifest parses the client component manifest JSON.
func LoadManifest(data []byte) (ClientManifest, error) {
	var m ClientManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return m, nil
}
