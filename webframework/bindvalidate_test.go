package webframework_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/polagonow/pola/core"
)

type bindInput struct {
	Name string `json:"name"`
}

// TestBindValidation proves c.Bind runs the installed validator uniformly on
// every adapter: a bind error yields 400, a validation error yields 422, and a
// valid body passes through to the handler. With no validator installed
// (TestConformance), Bind skips validation — so the two suites stay independent.
func TestBindValidation(t *testing.T) {
	// Install a validator the way validation.Plugin does; reset after.
	core.SetValidator(func(v any) error {
		if in, ok := v.(*bindInput); ok && in.Name == "" {
			return errors.New("name is required")
		}
		return nil
	})
	defer core.SetValidator(nil)

	for name, factory := range factories() {
		t.Run(name, func(t *testing.T) {
			fw := factory()
			fw.Handle("POST", "/v", func(c core.Context) error {
				var in bindInput
				if err := c.Bind(&in); err != nil {
					return err // Bind already wrote 400 (bind) or 422 (validation)
				}
				return c.JSON(http.StatusCreated, in)
			})
			h := fw.Handler()

			if w := request(h, "POST", "/v", `{"name":"ok"}`); w.Code != http.StatusCreated {
				t.Errorf("valid body: status = %d, want 201", w.Code)
			}
			if w := request(h, "POST", "/v", `{"name":""}`); w.Code != http.StatusUnprocessableEntity {
				t.Errorf("failed validation: status = %d, want 422", w.Code)
			}
			if w := request(h, "POST", "/v", `{bad json`); w.Code != http.StatusBadRequest {
				t.Errorf("bad json: status = %d, want 400", w.Code)
			}
		})
	}
}
