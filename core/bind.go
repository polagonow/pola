package core

import (
	"encoding"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
)

// defaultMaxMemory bounds in-memory multipart parsing for ShouldBindForm
// (matches net/http's own default).
const defaultMaxMemory = 32 << 20

// MustBind completes a "must"-style bind. Given the error returned by a
// ShouldBind* call, it writes a 400 JSON error response when err is non-nil
// (mirroring gin's Bind* abort) and returns err unchanged; on a nil error it is
// a no-op. Adapters compose it as core.MustBind(c, c.ShouldBindX(v)).
func MustBind(c Context, err error) error {
	if err != nil {
		_ = c.JSON(http.StatusBadRequest, M{"error": err.Error()})
	}
	return err
}

// ShouldBindUri maps path parameters into the struct pointed to by v using
// `uri:"name"` struct tags (the framework-neutral equivalent of gin's
// ShouldBindUri). Path params are single-valued.
//
// Binding does not validate: missing fields are left at their zero value. Run
// validation.Validate(v) afterward to enforce `validate:"..."` rules. For the
// must-variant that writes a 400 on failure, see MustBind.
func ShouldBindUri(c Context, v any) error {
	return bindStruct(v, "uri", func(name string) ([]string, bool) {
		if s := c.Param(name); s != "" {
			return []string{s}, true
		}
		return nil, false
	})
}

// ShouldBindHeader maps request headers into the struct pointed to by v using
// `header:"Name"` struct tags. Header names are matched case-insensitively, and
// repeated headers bind to slice fields ([]string, []int, ...).
func ShouldBindHeader(c Context, v any) error {
	h := c.Request().Header
	return bindStruct(v, "header", func(name string) ([]string, bool) {
		vs, ok := h[textproto.CanonicalMIMEHeaderKey(name)]
		return vs, ok
	})
}

// ShouldBindQuery maps URL query-string values into the struct pointed to by v
// using `query:"name"` struct tags. Repeated values (?id=1&id=2) bind to slices.
func ShouldBindQuery(c Context, v any) error {
	q := c.Request().URL.Query()
	return bindStruct(v, "query", func(name string) ([]string, bool) {
		vs, ok := q[name]
		return vs, ok
	})
}

// ShouldBindForm maps POST form-body values into the struct pointed to by v
// using `form:"name"` struct tags. Both urlencoded and multipart bodies are
// supported; file parts are ignored here (use FormFile/MultipartForm). Repeated
// values bind to slice fields. Query-string values are not consulted — use
// ShouldBindQuery for those.
func ShouldBindForm(c Context, v any) error {
	r := c.Request()
	// ParseMultipartForm calls ParseForm internally (populating r.PostForm for
	// urlencoded bodies) and only returns ErrNotMultipart for non-multipart
	// requests, which we ignore.
	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}
	return bindStruct(v, "form", func(name string) ([]string, bool) {
		if vs, ok := r.PostForm[name]; ok {
			return vs, true
		}
		if r.MultipartForm != nil {
			if vs, ok := r.MultipartForm.Value[name]; ok {
				return vs, true
			}
		}
		return nil, false
	})
}

// bindStruct populates the struct pointed to by v, mapping each exported field
// tagged `<tag>:"name"` to the value(s) returned by src(name). It converts
// strings to the field's type but performs no validation: absent values, and a
// single empty value for a scalar field, leave the field at its zero value.
func bindStruct(v any, tag string, src func(name string) (vals []string, ok bool)) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("bind: v must be a non-nil pointer to a struct, got %T", v)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("bind: v must point to a struct, got pointer to %s", rv.Kind())
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" { // unexported field
			continue
		}
		raw, ok := sf.Tag.Lookup(tag)
		if !ok {
			continue // untagged: no field-name fallback, leave at zero
		}
		name := raw
		if comma := strings.IndexByte(raw, ','); comma >= 0 {
			name = raw[:comma] // ignore options (e.g. ",omitempty")
		}
		if name == "" || name == "-" {
			continue
		}
		vals, present := src(name)
		if !present || len(vals) == 0 {
			continue
		}
		if err := setField(rv.Field(i), vals, name); err != nil {
			return err
		}
	}
	return nil
}

// setField assigns vals to the struct field fv. Slice fields (other than
// []byte) consume every value; scalar and pointer-to-scalar fields take the
// first. A single empty value for a scalar field is treated as absent.
func setField(fv reflect.Value, vals []string, name string) error {
	if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() != reflect.Uint8 {
		et := fv.Type().Elem()
		out := reflect.MakeSlice(fv.Type(), len(vals), len(vals))
		for i, s := range vals {
			if err := setScalar(out.Index(i), et, s, name); err != nil {
				return err
			}
		}
		fv.Set(out)
		return nil
	}
	if len(vals) == 1 && vals[0] == "" {
		return nil
	}
	return setScalar(fv, fv.Type(), vals[0], name)
}

// setScalar converts raw into the type ft and stores it in fv, allocating
// pointers as needed and honoring encoding.TextUnmarshaler.
func setScalar(fv reflect.Value, ft reflect.Type, raw, name string) error {
	if ft.Kind() == reflect.Pointer {
		p := reflect.New(ft.Elem())
		if err := setScalar(p.Elem(), ft.Elem(), raw, name); err != nil {
			return err
		}
		fv.Set(p)
		return nil
	}
	if fv.CanAddr() {
		if u, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if err := u.UnmarshalText([]byte(raw)); err != nil {
				return fmt.Errorf("invalid %s", name)
			}
			return nil
		}
	}
	switch ft.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid %s", name)
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, ft.Bits())
		if err != nil {
			return fmt.Errorf("invalid %s", name)
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, ft.Bits())
		if err != nil {
			return fmt.Errorf("invalid %s", name)
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, ft.Bits())
		if err != nil {
			return fmt.Errorf("invalid %s", name)
		}
		fv.SetFloat(f)
	default:
		return fmt.Errorf("bind %s: unsupported field type %s", name, ft.Kind())
	}
	return nil
}
