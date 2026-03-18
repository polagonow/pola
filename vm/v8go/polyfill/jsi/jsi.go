// Package jsi installs the __JSI__ bridge object for the V8 VM.
//
// __JSI__ is an empty object populated at request time with async Go bridge
// functions. It is cleared between requests by ClearState.
package jsi

import (
	"fmt"

	v8 "rogchap.com/v8go"

	"github.com/polagonow/pola/framework/globals"
)

// Enable installs __JSI__ as an empty global object into ctx.
func Enable(ctx *v8.Context) error {
	if _, err := ctx.RunScript("globalThis."+globals.BridgeObject+" = {};", "jsi.js"); err != nil {
		return fmt.Errorf("jsi: %w", err)
	}
	return nil
}
