package request

import (
	"encoding/json"
	"fmt"

	"github.com/polagonow/pola/core"
)

// DecodeMultipartJSON parses a multipart form on the request and decodes the
// JSON "data" field into v. Non-file fields are sent as a JSON blob in the
// "data" form field; files are sent as separate multipart file parts (read them
// via c.FormFile or c.MultipartForm).
func DecodeMultipartJSON(c core.Context, v any, maxMemory int64) error {
	if err := c.Request().ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("parse multipart form: %w", err)
	}
	data := c.FormValue("data")
	if data == "" {
		return fmt.Errorf("missing 'data' field in multipart form")
	}
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return fmt.Errorf("invalid JSON in 'data' field: %w", err)
	}
	return nil
}
