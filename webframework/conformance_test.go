package webframework_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/chi"
	"github.com/polagonow/pola/webframework/echo"
	"github.com/polagonow/pola/webframework/gin"
	"github.com/polagonow/pola/webframework/std"
)

// factories returns one constructor per adapter so the same conformance suite
// runs against all of them.
func factories() map[string]func() core.WebFramework {
	return map[string]func() core.WebFramework{
		"std":  func() core.WebFramework { return std.New() },
		"gin":  func() core.WebFramework { return gin.New() },
		"echo": func() core.WebFramework { return echo.New() },
		"chi":  func() core.WebFramework { return chi.New() },
	}
}

// build wires a fixed set of routes/mounts/middleware/fallback onto fw and
// returns the root handler.
func build(fw core.WebFramework) http.Handler {
	fw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW", "1")
			next.ServeHTTP(w, r)
		})
	})
	fw.Mount("/_pola/x", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("MOUNT"))
	}))
	fw.Handle("GET", "/ping", func(c core.Context) error { return c.JSON(200, core.M{"msg": "pong"}) })
	fw.Handle("GET", "/users/:id", func(c core.Context) error { return c.String(200, "id="+c.Param("id")) })
	fw.Handle("GET", "/legacy/:id", func(c core.Context) error {
		return c.String(200, "legacy="+core.Param(c.Request(), "id"))
	})
	fw.Handle("POST", "/items", func(c core.Context) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&in); err != nil {
			return c.JSON(400, core.M{"error": "bad"})
		}
		return c.JSON(201, core.M{"name": in.Name})
	})
	// Bind all four sources into one struct via the core.Context methods. Driven
	// against every adapter, this proves the shared binder behaves identically no
	// matter which engine populates Param/URL/Header/Body.
	fw.Handle("POST", "/bind/:id", func(c core.Context) error {
		var in struct {
			ID    int    `uri:"id"`
			Page  int    `query:"page"`
			Token string `header:"X-Token"`
			Email string `form:"email"`
		}
		for _, bind := range []func(any) error{c.ShouldBindUri, c.ShouldBindQuery, c.ShouldBindHeader, c.ShouldBindForm} {
			if err := bind(&in); err != nil {
				return c.JSON(400, core.M{"error": err.Error()})
			}
		}
		return c.String(200, fmt.Sprintf("id=%d page=%d token=%s email=%s", in.ID, in.Page, in.Token, in.Email))
	})
	// Must-variant: on a bad value it writes the 400 itself, so the handler just
	// returns the error. Proves the auto-400 behaves identically on every adapter.
	fw.Handle("GET", "/mustbind", func(c core.Context) error {
		var in struct {
			N int `query:"n"`
		}
		if err := c.BindQuery(&in); err != nil {
			return err
		}
		return c.JSON(200, core.M{"n": in.N})
	})
	fw.Handle("GET", "/boom", func(core.Context) error { return errors.New("boom") })
	fw.Fallback(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("FALLBACK"))
	}))
	return fw.Handler()
}

func request(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestConformance(t *testing.T) {
	for name, factory := range factories() {
		t.Run(name, func(t *testing.T) {
			h := build(factory())

			t.Run("json", func(t *testing.T) {
				w := request(h, "GET", "/ping", "")
				if w.Code != 200 || !strings.Contains(w.Body.String(), "pong") {
					t.Fatalf("GET /ping = %d %q", w.Code, w.Body.String())
				}
			})
			t.Run("middleware", func(t *testing.T) {
				if w := request(h, "GET", "/ping", ""); w.Header().Get("X-MW") != "1" {
					t.Errorf("expected X-MW header from middleware")
				}
			})
			t.Run("param", func(t *testing.T) {
				if w := request(h, "GET", "/users/42", ""); w.Body.String() != "id=42" {
					t.Errorf("GET /users/42 = %q, want id=42", w.Body.String())
				}
			})
			t.Run("legacy_param", func(t *testing.T) {
				if w := request(h, "GET", "/legacy/7", ""); w.Body.String() != "legacy=7" {
					t.Errorf("GET /legacy/7 = %q, want legacy=7", w.Body.String())
				}
			})
			t.Run("bind", func(t *testing.T) {
				w := request(h, "POST", "/items", `{"name":"alice"}`)
				if w.Code != 201 || !strings.Contains(w.Body.String(), "alice") {
					t.Errorf("POST /items = %d %q", w.Code, w.Body.String())
				}
			})
			t.Run("bind_struct", func(t *testing.T) {
				r := httptest.NewRequest("POST", "/bind/42?page=3", strings.NewReader("email=a@b.com"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				r.Header.Set("X-Token", "secret")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != 200 || w.Body.String() != "id=42 page=3 token=secret email=a@b.com" {
					t.Errorf("POST /bind/42 = %d %q, want 200 id=42 page=3 token=secret email=a@b.com", w.Code, w.Body.String())
				}
			})
			t.Run("must_bind_400", func(t *testing.T) {
				w := request(h, "GET", "/mustbind?n=notanumber", "")
				if w.Code != 400 || !strings.Contains(w.Body.String(), "invalid n") {
					t.Errorf("GET /mustbind = %d %q, want 400 containing 'invalid n'", w.Code, w.Body.String())
				}
			})
			t.Run("mount", func(t *testing.T) {
				if w := request(h, "GET", "/_pola/x", ""); w.Body.String() != "MOUNT" {
					t.Errorf("GET /_pola/x = %q, want MOUNT", w.Body.String())
				}
			})
			t.Run("fallback", func(t *testing.T) {
				w := request(h, "GET", "/does/not/exist", "")
				if w.Code != 404 || w.Body.String() != "FALLBACK" {
					t.Errorf("GET /does/not/exist = %d %q, want 404 FALLBACK", w.Code, w.Body.String())
				}
			})
			t.Run("error_to_500", func(t *testing.T) {
				if w := request(h, "GET", "/boom", ""); w.Code < 500 {
					t.Errorf("GET /boom = %d, want >= 500", w.Code)
				}
			})
		})
	}
}
