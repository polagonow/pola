package tests

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRSC_Concurrent(t *testing.T) {
	app := requireApp(t)
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
				results <- fmt.Errorf("bad output: %s", string(body)[:min(len(string(body)), 100)])
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
}
