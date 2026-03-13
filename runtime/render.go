package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
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

// Render calls __render__ in the VM, pulls the RSC Flight ReadableStream, and
// writes all chunks to fw.
//
// react-server-dom-webpack/server.browser returns the ReadableStream
// synchronously (no Promise), so we call __render__ then immediately pull
// chunks in a loop via __pullStream__.
//
// Each __pullStream__ call:
//   1. Calls stream._start()  (runs start(controller) once)
//   2. Drains our microtask queue (scheduleWork callbacks enqueue chunks)
//   3. Calls stream._pull()   (may enqueue more chunks)
//   4. Drains microtask queue again
//   5. Returns { chunks: [...], done: bool }
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

	// renderToReadableStream returns the stream synchronously
	if _, err := vm.RunString(fmt.Sprintf(
		`__gojsx_stream__ = __render__(%q, %q)`,
		exportName, string(propsJSON),
	)); err != nil {
		return fmt.Errorf("render: __render__: %w", err)
	}

	// Collect all Flight chunks
	flightVal, err := vm.RunString(`(function () {
		var dec = new TextDecoder();
		var out = "";
		var safety = 0;
		while (safety++ < 2000) {
			var r = __pullStream__(__gojsx_stream__);
			for (var i = 0; i < r.chunks.length; i++) {
				out += dec.decode(r.chunks[i]);
			}
			if (r.done) break;
		}
		return out;
	})()`)
	if err != nil {
		return fmt.Errorf("render: pull stream: %w", err)
	}

	flight := flightVal.String()
	if flight == "" {
		return fmt.Errorf("render: empty Flight output")
	}

	if _, err := fw.WriteRaw([]byte(flight)); err != nil {
		return fmt.Errorf("render: write: %w", err)
	}
	fw.Flush()

	// Clean up for next request
	vm.RunString(`__gojsx_stream__ = null`)

	return nil
}

// renderFallback is the manual tree-walker used when __render__ is absent.
func (r *Renderer) renderFallback(fw *FlightWriter, vm *VM, opts RenderOptions, propsJSON []byte) error {
	props := map[string]any{}
	_ = json.Unmarshal(propsJSON, &props)

	sched := NewSuspenseScheduler(fw)
	rootVal, err := vm.CallExport(opts.ExportName, vm.rt.ToValue(props))
	if err != nil {
		return fmt.Errorf("render fallback: call %s: %w", opts.ExportName, err)
	}

	shellID := fw.NextID()
	shellNode, err := walkNode(vm.rt, rootVal, fw, sched, r.manifest)
	if err != nil {
		return fmt.Errorf("render fallback: walk: %w", err)
	}
	if err := fw.WriteChunk(shellID, ChunkJSON, shellNode); err != nil {
		return err
	}
	sched.FlushAll()
	for _, ref := range r.manifest {
		fw.WriteModuleRef(fw.NextID(), ref)
	}
	return nil
}

// ClientManifest maps component IDs to their client bundle references.
type ClientManifest map[string]ClientRef

// LoadManifest parses the client component manifest JSON.
func LoadManifest(data []byte) (ClientManifest, error) {
	var m ClientManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	return m, nil
}
