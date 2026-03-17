// Package vm provides shared type aliases for the GoJSX VM layer.
// Concrete VM implementations live in sub-packages (e.g. vm/goja).
package vm

import (
	_ "gojsx/vm/goja"
	// _ "gojsx/vm/quickjsgo"
	// _ "gojsx/vm/quickjsgo"
	// _ "gojsx/vm/v8go"
	// _ "gojsx/vm/sobek"
	// _ "gojsx/vm/moderncquickjs"
	// _ "gojsx/vm/qjs"
)
