package suite

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// RunConcurrentRenderingTests verifies that the server handles multiple
// simultaneous RSC requests without data races or corrupted output.
func RunConcurrentRenderingTests(t *testing.T) {
	t.Helper()

	t.Run("SimultaneousRequestsAllSucceed", func(t *testing.T) {
		fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
			app := f.GetApp(t)
			const n = 10
			results := make(chan error, n)
			for range n {
				go func() {
					req := httptest.NewRequestWithContext(context.Background(), "GET", "/posts", nil)
					req.Header.Set("Content-Type", "text/x-component")
					w := httptest.NewRecorder()
					app.ServeHTTP(w, req)
					body, _ := io.ReadAll(w.Result().Body)
					if !strings.Contains(string(body), "0:") {
						results <- fmt.Errorf("concurrent request produced invalid Flight output: %s",
							string(body)[:min(len(string(body)), 100)])
					} else {
						results <- nil
					}
				}()
			}
			for range n {
				if err := <-results; err != nil {
					t.Error(err)
				}
			}
		})
	})
}
