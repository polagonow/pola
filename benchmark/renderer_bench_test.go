package benchmark_test

import "testing"

func BenchmarkReactRenderPlaceholder(b *testing.B) {
	b.Skip("requires full build pipeline - run with: POLA_VM=goja POLA_RENDERER=react mage benchmark")
}
