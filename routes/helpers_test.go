package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusOK, map[string]string{"name": "test"})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "test" {
		t.Fatalf("expected name=test, got %q", body["name"])
	}
}

func TestWriteJSON_StatusCreated(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusCreated, map[string]int{"id": 1})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "bad input")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "bad input" {
		t.Fatalf("expected error='bad input', got %q", body["error"])
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}`))
	var p payload
	if err := DecodeJSON(req, &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "alice" {
		t.Fatalf("expected alice, got %q", p.Name)
	}
}

func TestDecode_InvalidWriteJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid`))
	var p struct{}
	if err := DecodeJSON(req, &p); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func withParams(r *http.Request, params map[string]any) *http.Request {
	ctx := context.WithValue(r.Context(), paramsKey, params)
	return r.WithContext(ctx)
}

func TestIDParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withParams(req, map[string]any{"id": "abc"})

	id, ok := IDParam(req)
	if !ok || id != "abc" {
		t.Fatalf("expected (abc, true), got (%q, %v)", id, ok)
	}
}

func TestIDParam_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, ok := IDParam(req)
	if ok {
		t.Fatal("expected false for missing id param")
	}
}

func TestParseUintID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withParams(req, map[string]any{"id": "42"})

	n, err := ParseUintID(req)
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
}

func TestParseUintID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := ParseUintID(req)
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestParseUintID_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withParams(req, map[string]any{"id": "abc"})

	_, err := ParseUintID(req)
	if err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestQueryInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=3", nil)
	if n := QueryInt(req, "page", 1); n != 3 {
		t.Fatalf("expected 3, got %d", n)
	}
}

func TestQueryInt_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if n := QueryInt(req, "page", 1); n != 1 {
		t.Fatalf("expected fallback 1, got %d", n)
	}
}

func TestQueryInt_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)
	if n := QueryInt(req, "page", 1); n != 1 {
		t.Fatalf("expected fallback 1, got %d", n)
	}
}

func TestQueryInt_Zero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=0", nil)
	if n := QueryInt(req, "page", 1); n != 1 {
		t.Fatalf("expected fallback 1 for zero value, got %d", n)
	}
}
