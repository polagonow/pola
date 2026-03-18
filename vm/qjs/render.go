package qjs

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/framework/globals"

	qjs "github.com/fastschema/qjs"
)

var (
	renderScriptTmpl  = template.Must(template.New("renderScript").Parse("{{.RenderAsyncFn}}({{.ExportLit}}, {{.PropsLit}})"))
	clearOutputTmpl   = template.Must(template.New("clearOutput").Parse("{{.OutputChunk}} = undefined;"))
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
// __renderAsync__(exportName, propsJSON) and awaits the returned Promise.
//
// Because bridge functions are synchronous (they block the host-function call
// while a Go goroutine runs), all WASM operations happen sequentially on the
// same goroutine. The mutex is held for the full duration to prevent concurrent
// use from another request.
func DrainStream(vm *VM, w framework.StreamWriter, sess *RenderSession) (wroteAny bool, err error) {
	var runErr error

	vm.run(func() {
		// Install the per-request output sink. Called synchronously from JS
		// during promise.Await(), so w is written incrementally.
		vm.ctx.SetFunc(globals.OutputChunk, func(this *qjs.This) (*qjs.Value, error) {
			args := this.Args()
			if len(args) > 0 && args[0].IsString() {
				if chunk := args[0].String(); len(chunk) > 0 {
					w.WriteRaw([]byte(chunk)) //nolint:errcheck
					w.Flush()
					wroteAny = true
				}
			}
			return this.Context().NewUndefined(), nil
		})

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

		promise, evalErr := vm.ctx.Eval("render.js", qjs.Code(script))
		if evalErr != nil {
			runErr = fmt.Errorf("qjs: render eval: %w", evalErr)
			return
		}
		defer promise.Free()

		// Await drives the QuickJS job loop until __renderAsync__ resolves.
		// Bridge host-functions block inline (no goroutines call WASM), so
		// promise.Await() and all bridge calls share the same goroutine safely.
		result, awaitErr := promise.Await()
		if result != nil {
			defer result.Free()
		}
		if awaitErr != nil {
			runErr = fmt.Errorf("qjs: render await: %w", awaitErr)
			return
		}

		// Release the closure's reference to w so the pooled VM doesn't retain it.
		var clearBuf strings.Builder
		_ = clearOutputTmpl.Execute(&clearBuf, struct{ OutputChunk string }{globals.OutputChunk})
		_, _ = vm.ctx.Eval("clear_output.js", qjs.Code(clearBuf.String()))
	})

	return wroteAny, runErr
}
