package quickjsgo

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	quickjs "github.com/buke/quickjs-go"

	"gojsx/framework"
	"gojsx/framework/globals"
)

var (
	renderScriptTmpl = template.Must(template.New("renderScript").Parse("{{.RenderAsyncFn}}({{.ExportLit}}, {{.PropsLit}})"))
	clearOutputTmpl  = template.Must(template.New("clearOutput").Parse("{{.OutputChunk}} = undefined;"))
)

// RenderSession holds the parameters for a single render.
// The actual JS execution happens lazily in DrainStream.
type RenderSession struct {
	ExportName string
	PropsJSON  string
}

// StartRender stores the render parameters. The JS call happens in DrainStream.
func StartRender(_ *VM, exportName, propsJSON string) (*RenderSession, error) {
	return &RenderSession{ExportName: exportName, PropsJSON: propsJSON}, nil
}

// DrainStream installs a per-request __outputChunk__ Go function, then calls
// __renderAsync__(exportName, propsJSON) via ctx.Await. Each time JS produces a
// chunk it calls __outputChunk__ synchronously — because Go-backed functions run
// on the JS thread during ctx.Await, this flushes each chunk to w immediately,
// enabling React Suspense streaming.
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	var runErr error

	vm.run(func() {
		// Install the per-request output sink. __outputChunk__ is called
		// synchronously from JS during ctx.Await, so w is written incrementally.
		outputFn := vm.ctx.NewFunction(func(qCtx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value { //nolint:lll
			if len(args) > 0 && args[0].IsString() {
				if chunk := args[0].String(); len(chunk) > 0 {
					w.WriteRaw([]byte(chunk)) //nolint:errcheck
					w.Flush()
					wroteAny = true
				}
			}
			return qCtx.NewUndefined()
		})
		g := vm.ctx.Globals()
		g.Set(globals.OutputChunk, outputFn) // ownership transferred; do NOT free outputFn or g

		// Evaluate the async render — returns a pending Promise.
		exportLit, _ := json.Marshal(sess.ExportName)
		propsLit, _ := json.Marshal(sess.PropsJSON)
		var scriptBuf strings.Builder
		_ = renderScriptTmpl.Execute(&scriptBuf, struct {
			RenderAsyncFn string
			ExportLit     string
			PropsLit      string
		}{globals.RenderAsyncFn, string(exportLit), string(propsLit)})
		script := scriptBuf.String()

		promise := vm.ctx.Eval(script, quickjs.EvalFileName("render.js"))
		defer promise.Free()

		if promise.IsException() {
			runErr = fmt.Errorf("quickjsgo: render eval: %w", vm.ctx.Exception())
			return
		}

		// ctx.Await drives the event loop:
		//   - JS_ExecutePendingJob  — runs native Promise continuations (async components)
		//   - ctx.ProcessJobs()     — runs Go-scheduled bridge-function callbacks
		// During each iteration, __renderAsync__ calls __outputChunk__ synchronously,
		// flushing each chunk to w before the Promise fully resolves.
		result := vm.ctx.Await(promise)
		defer result.Free()

		if result.IsException() {
			runErr = fmt.Errorf("quickjsgo: render await: %w", vm.ctx.Exception())
			return
		}

		// Release the closure's reference to w so the pool VM doesn't retain it.
		var clearBuf strings.Builder
		_ = clearOutputTmpl.Execute(&clearBuf, struct{ OutputChunk string }{globals.OutputChunk})
		clearRet := vm.ctx.Eval(clearBuf.String(), quickjs.EvalFileName("clear_output.js"))
		clearRet.Free()
	})

	return wroteAny, runErr
}
