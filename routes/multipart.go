package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DecodeMultipartJSON parses a multipart form and decodes the JSON "data" field into v.
// Non-file fields are sent as a JSON blob in the "data" form field; files are sent
// as separate multipart file parts.
func DecodeMultipartJSON(r *http.Request, v any, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("parse multipart form: %w", err)
	}
	data := r.FormValue("data")
	if data == "" {
		return fmt.Errorf("missing 'data' field in multipart form")
	}
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return fmt.Errorf("invalid JSON in 'data' field: %w", err)
	}
	return nil
}
