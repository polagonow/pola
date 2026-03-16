// Package bundler registers the default esbuild bundler implementation.
// Importing this package is sufficient to wire up esbuild — no need to import
// gojsx/bundler/esbuild directly.
package bundler

import _ "gojsx/bundler/esbuild"
