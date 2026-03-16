// Package vm provides shared type aliases for the GoJSX VM layer.
// Concrete VM implementations live in sub-packages (e.g. vm/goja).
package vm

import (
	_ "gojsx/vm/goja"
	// _ "gojsx/vm/v8go"
)
