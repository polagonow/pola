// Package console provides a console global for the V8 VM.
//
// It installs a Go-backed __v8_log__ function, then wires console.log/warn/
// error/info to it via a small JS snippet.
package console

import (
	"fmt"
	"strings"

	v8 "rogchap.com/v8go"
)

const src = `
globalThis.console = {
	log:   function() { __v8_log__("LOG",  Array.prototype.slice.call(arguments)); },
	warn:  function() { __v8_log__("WARN", Array.prototype.slice.call(arguments)); },
	error: function() { __v8_log__("ERR",  Array.prototype.slice.call(arguments)); },
	info:  function() { __v8_log__("INFO", Array.prototype.slice.call(arguments)); },
};
`

// Enable installs the console global into ctx.
func Enable(iso *v8.Isolate, ctx *v8.Context) error {
	logFT := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) < 2 {
			return nil
		}
		level := args[0].String()
		arr := args[1].Object()
		lenVal, _ := arr.Get("length")
		n := int(lenVal.Integer())
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			v, _ := arr.GetIdx(uint32(i))
			parts[i] = v.String()
		}
		fmt.Printf("[VM:%s] %s\n", level, strings.Join(parts, " "))
		return nil
	})
	if err := ctx.Global().Set("__v8_log__", logFT.GetFunction(ctx)); err != nil {
		return fmt.Errorf("console: set __v8_log__: %w", err)
	}
	if _, err := ctx.RunScript(src, "console.js"); err != nil {
		return fmt.Errorf("console: %w", err)
	}
	return nil
}
