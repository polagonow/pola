package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("routes.WriteJSON: write error: %v", err)
	}
}

// WriteError writes a JSON error envelope: {"error": "message"}.
func WriteError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := fmt.Fprintf(w, `{"error":%q}`, msg); err != nil {
		log.Printf("routes.WriteError: write error: %v", err)
	}
}

// DecodeJSON reads JSON from the request body into v.
func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// IDParam returns the "id" URL parameter as a string.
// Returns ("", false) if the parameter is empty.
func IDParam(r *http.Request) (string, bool) {
	id := Param(r, "id")
	if id == "" {
		return "", false
	}
	return id, true
}

// ParseUintID extracts and parses the "id" URL parameter as a uint.
func ParseUintID(r *http.Request) (uint, error) {
	id := Param(r, "id")
	if id == "" {
		return 0, fmt.Errorf("missing id")
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id")
	}
	return uint(n), nil
}

// QueryInt reads an integer query parameter with a default fallback.
func QueryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
