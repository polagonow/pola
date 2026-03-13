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

	var wroteAny bool
	var renderErr error

	err = vm.run(func(rt *goja.Runtime) error {
		// __gojsx_chunk__ is called from JS for each batch of Flight bytes.
		// It writes and flushes immediately so the client sees chunks as they
		// arrive — the Suspense fallback is flushed on the very first poll
		// tick, resolved content streams in after async components finish.
		rt.Set("__gojsx_chunk__", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				fw.WriteRaw([]byte(call.Arguments[0].String())) //nolint:errcheck
				fw.Flush()
				wroteAny = true
			}
			return goja.Undefined()
		})

		// Start the render and schedule the first poll tick.
		// Each __poll__ call drains one batch of stream chunks and flushes
		// them immediately. If the stream is not done it reschedules itself
		// via setTimeout(0), yielding to the event loop so that native
		// Promises (async/await in server components) can resolve between
		// ticks. loop.Run returns once no more timers are pending.
		script := fmt.Sprintf(`(function() {
	__gojsx_stream__ = __render__(%q, %q);
	var dec = new TextDecoder();
	function __poll__() {
		var r = __pullStream__(__gojsx_stream__);
		if (r.chunks.length > 0) {
			var text = "";
			for (var i = 0; i < r.chunks.length; i++) {
				text += dec.decode(r.chunks[i]);
			}
			__gojsx_chunk__(text);
		}
		if (!r.done) {
			// Back off 5ms when there are no chunks so we don't spin-burn CPU
			// while waiting for async server components (Promises) to resolve.
			// When the goroutine calls RunOnLoop(resolve), r.leave() drains
			// native microtasks so React renders the component and enqueues
			// chunks before the next poll tick fires.
			setTimeout(__poll__, r.chunks.length > 0 ? 0 : 5);
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
	if !wroteAny {
		return fmt.Errorf("render: no Flight output written")
	}

	// Clean up per-request globals for next request.
	_ = vm.run(func(rt *goja.Runtime) error {
		rt.Set("__gojsx_stream__", goja.Undefined())
		rt.Set("__gojsx_chunk__", goja.Undefined())
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
