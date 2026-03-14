package runtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
// a Go for loop, and writes all chunks to fw.
//
// Each iteration calls vm.run (loop.Run), which drains all pending timers and
// RunOnLoop callbacks before returning. This lets goroutine-driven async/await
// in server components resolve between iterations without any setTimeout scheduling.
func (r *Renderer) Render(fw *FlightWriter, vm *VM, opts RenderOptions) error {
	if err := vm.SetRequestContext(opts.RequestContext); err != nil {
		return fmt.Errorf("render: set context: %w", err)
	}
	propsJSON, err := json.Marshal(opts.Props)
	if err != nil {
		return fmt.Errorf("render: marshal props: %w", err)
	}

	sess, err := startRender(vm, opts.ExportName, string(propsJSON))
	if err != nil {
		return fmt.Errorf("render: setup: %w", err)
	}

	wroteAny, err := drainStream(vm, fw, sess)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if !wroteAny {
		return fmt.Errorf("render: no Flight output written")
	}
	return nil
}

// renderSession holds the JS callables and stream value for a single render.
// All fields are only valid inside a vm.run callback.
type renderSession struct {
	pullStreamFn goja.Callable
	decoderObj   *goja.Object
	decodeFn     goja.Callable
	stream       goja.Value
}

// startRender looks up __render__ and __pullStream__, instantiates a TextDecoder,
// and calls __render__ to obtain the RSC Flight ReadableStream.
func startRender(vm *VM, exportName, propsJSON string) (*renderSession, error) {
	var sess renderSession
	err := vm.run(func(rt *goja.Runtime) error {
		renderFn, ok := goja.AssertFunction(rt.Get("__render__"))
		if !ok {
			return fmt.Errorf("__render__ is not a function")
		}
		sess.pullStreamFn, ok = goja.AssertFunction(rt.Get("__pullStream__"))
		if !ok {
			return fmt.Errorf("__pullStream__ is not a function")
		}

		decoderCtor, ok := goja.AssertConstructor(rt.Get("TextDecoder"))
		if !ok {
			return fmt.Errorf("TextDecoder is not a constructor")
		}
		var err error
		sess.decoderObj, err = decoderCtor(rt.NewObject())
		if err != nil {
			return err
		}
		sess.decodeFn, ok = goja.AssertFunction(sess.decoderObj.Get("decode"))
		if !ok {
			return fmt.Errorf("TextDecoder.decode is not a function")
		}

		sess.stream, err = renderFn(goja.Undefined(), rt.ToValue(exportName), rt.ToValue(propsJSON))
		return err
	})
	return &sess, err
}

// drainStream polls sess until the stream is done, writing decoded chunks to fw.
// Each vm.run call drains all pending jobs (including goroutine-scheduled Promise
// resolutions via RunOnLoop) before returning, so async server components resolve
// naturally between iterations.
func drainStream(vm *VM, fw *FlightWriter, sess *renderSession) (wroteAny bool, err error) {
	for {
		var done, noChunks bool
		if err := vm.run(func(rt *goja.Runtime) error {
			r, err := sess.pullStreamFn(goja.Undefined(), sess.stream)
			if err != nil {
				return err
			}
			rObj := r.ToObject(rt)
			chunksArr := rObj.Get("chunks").ToObject(rt)
			chunksLen := int(chunksArr.Get("length").ToInteger())
			if chunksLen > 0 {
				var sb strings.Builder
				for i := range chunksLen {
					decoded, err := sess.decodeFn(sess.decoderObj, chunksArr.Get(strconv.Itoa(i)))
					if err != nil {
						return err
					}
					sb.WriteString(decoded.String())
				}
				fw.WriteRaw([]byte(sb.String())) //nolint:errcheck
				fw.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			done = rObj.Get("done").ToBoolean()
			return nil
		}); err != nil {
			return wroteAny, err
		}
		if done {
			break
		}
		if noChunks {
			// Back off when idle so we don't spin-burn CPU while waiting for
			// goroutine-driven Promise resolution. The event loop is not
			// blocked during this sleep.
			time.Sleep(5 * time.Millisecond)
		}
	}
	return wroteAny, nil
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
