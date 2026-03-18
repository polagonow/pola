package v8go

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/framework/globals"
)

var (
	startRenderScriptTmpl = template.Must(template.New("startRender").Parse(
		"globalThis.{{.StreamHandle}} = {{.RenderFn}}({{.ExportNameLit}}, {{.PropsJSONLit}});"))
	pullScriptTmpl = template.Must(template.New("pull").Parse("{{.PullStreamFn}}({{.StreamHandle}})"))
)

// V8RenderSession holds per-render state. The stream is stored in the
// stream-handle global so __pullStream__ can access it by name.
type V8RenderSession struct{}

// StartRender calls __render__(exportName, propsJSON) and stores the returned
// ReadableStream in the stream-handle global for DrainStream to consume.
func StartRender(vm *V8VM, exportName, propsJSON string) (*V8RenderSession, error) {
	var sess V8RenderSession
	err := vm.run(func() error {
		exportNameLit, _ := json.Marshal(exportName)
		propsJSONLit, _ := json.Marshal(propsJSON)
		var scriptBuf strings.Builder
		_ = startRenderScriptTmpl.Execute(&scriptBuf, struct {
			StreamHandle, RenderFn string
			ExportNameLit, PropsJSONLit string
		}{globals.StreamHandle, globals.RenderFn, string(exportNameLit), string(propsJSONLit)})
		_, err := vm.ctx.RunScript(scriptBuf.String(), "render.js")
		return err
	})
	return &sess, err
}

// DrainStream polls the stream until done, writing decoded chunks to w.
// This mirrors the polling pattern in vm/goja/render.go.
func DrainStream(vm *V8VM, w framework.StreamWriter, _ *V8RenderSession) (wroteAny bool, err error) {
	for {
		var done, noChunks bool
		if runErr := vm.run(func() error {
			var pullBuf strings.Builder
			_ = pullScriptTmpl.Execute(&pullBuf, struct{ PullStreamFn, StreamHandle string }{globals.PullStreamFn, globals.StreamHandle})
			result, err := vm.ctx.RunScript(pullBuf.String(), "pull.js")
			if err != nil {
				return fmt.Errorf("v8: pull stream: %w", err)
			}
			resultObj := result.Object()

			chunksVal, err := resultObj.Get("chunks")
			if err != nil {
				return fmt.Errorf("v8: get chunks: %w", err)
			}
			doneVal, err := resultObj.Get("done")
			if err != nil {
				return fmt.Errorf("v8: get done: %w", err)
			}
			done = doneVal.Boolean()

			chunksObj := chunksVal.Object()
			lenVal, _ := chunksObj.Get("length")
			chunksLen := int(lenVal.Integer())

			if chunksLen > 0 {
				var sb strings.Builder
				for i := 0; i < chunksLen; i++ {
					chunkVal, err := chunksObj.GetIdx(uint32(i))
					if err != nil {
						continue
					}
					// Decode Uint8Array bytes. Each chunk is a Uint8Array produced by
					// React Flight's TextEncoder.encode(); read byte-by-byte via Object.GetIdx.
					if chunkVal.IsUint8Array() {
						sb.Write(readUint8Array(chunkVal.Object()))
					} else {
						sb.WriteString(chunkVal.String())
					}
				}
				w.WriteRaw([]byte(sb.String())) //nolint:errcheck
				w.Flush()
				wroteAny = true
			} else {
				noChunks = true
			}
			return nil
		}); runErr != nil {
			return wroteAny, runErr
		}

		if done {
			break
		}
		if noChunks {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return wroteAny, nil
}
