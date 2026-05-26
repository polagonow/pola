package benchmark_test

import "testing"

func BenchmarkEsbuildBundlePlaceholder(b *testing.B) {
	b.Skip("requires full build pipeline - run with: POLA_BUNDLER=esbuild POLA_RENDERER=react mage benchmark")
}
