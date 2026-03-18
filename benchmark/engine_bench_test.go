package benchmark_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/engine/goja"
)

func BenchmarkGojaEngineEval(b *testing.B) {
	eng, err := goja.New("(function() { return 42; })()")
	if err != nil {
		b.Skip("goja unavailable:", err)
	}
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	defer rt.Dispose()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rt.Eval("1 + 1")
		if err != nil {
			b.Fatal(err)
		}
	}
}
