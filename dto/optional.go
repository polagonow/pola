package dto

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// Optional wraps a value of type T while recording whether that value was
// actually supplied. It is the piece that makes partial-update (HTTP PATCH)
// semantics correct: for a plain `Price float64` field there is no way to tell
// "the client sent price: 0" apart from "the client omitted price entirely" —
// both arrive as the zero value. Optional keeps that distinction without
// resorting to pointers (which conflate "absent" with "explicit null" and are
// awkward to dereference).
//
// A field declared `Price Optional[float64]` is:
//   - absent      → Present == false (left untouched by [Copy])
//   - present     → Present == true, Val holds the decoded value (including 0)
//
// Declare optional DTO fields with the `omitzero` JSON option so absent values
// disappear from responses instead of serializing as null:
//
//	type UpdateProduct struct {
//	    Name  dto.Optional[string]  `json:"name,omitzero"`
//	    Price dto.Optional[float64] `json:"price,omitzero"`
//	}
//
// Optional implements [json.Marshaler]/[json.Unmarshaler], [driver.Valuer] and
// [sql.Scanner] (via the standard library's generic [sql.Null]), and the
// unexported presence protocol read by [Copy], so it round-trips cleanly through
// request bodies, database columns and the DTO→model mapper alike.
type Optional[T any] struct {
	Val     T
	Present bool
}

// NewOptional returns a present Optional holding v.
func NewOptional[T any](v T) Optional[T] { return Optional[T]{Val: v, Present: true} }

// Set stores v and marks the field present.
func (o *Optional[T]) Set(v T) {
	o.Val = v
	o.Present = true
}

// Unset clears the value and marks the field absent.
func (o *Optional[T]) Unset() {
	var zero T
	o.Val = zero
	o.Present = false
}

// IsPresent reports whether a value was supplied.
func (o Optional[T]) IsPresent() bool { return o.Present }

// IsZero reports whether the field is absent. It exists so the encoding/json
// `omitzero` option (Go 1.24+) drops unset optionals during marshaling.
func (o Optional[T]) IsZero() bool { return !o.Present }

// Get returns the value and whether it is present, mirroring the comma-ok idiom.
func (o Optional[T]) Get() (T, bool) { return o.Val, o.Present }

// Default returns the value when present, otherwise def.
func (o Optional[T]) Default(def T) T {
	if o.Present {
		return o.Val
	}
	return def
}

// value implements the unexported presence protocol consumed by [Copy], letting
// the mapper read an Optional's payload without knowing T.
func (o Optional[T]) value() any { return o.Val }

// MarshalJSON emits the wrapped value, or null when absent. Pair the field with
// the `,omitzero` tag option so absent optionals are omitted entirely rather
// than rendered as null.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Present {
		return []byte("null"), nil
	}
	return json.Marshal(o.Val)
}

// UnmarshalJSON records the field as present and decodes the value. Any key in
// the payload counts as present — including an explicit null, which decodes to
// the zero value while still reporting IsPresent() == true.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		var zero T
		o.Val = zero
		return nil
	}
	return json.Unmarshal(data, &o.Val)
}

// Value implements driver.Valuer: an absent optional is a SQL NULL; a present
// one delegates to the standard library's generic sql.Null[T], which applies
// the usual driver conversions (and honors driver.Valuer on T).
func (o Optional[T]) Value() (driver.Value, error) {
	if !o.Present {
		return nil, nil
	}
	return sql.Null[T]{V: o.Val, Valid: true}.Value()
}

// Scan implements sql.Scanner, delegating to sql.Null[T]. A NULL column yields
// an absent optional; any other value marks the field present.
func (o *Optional[T]) Scan(src any) error {
	var n sql.Null[T]
	if err := n.Scan(src); err != nil {
		return err
	}
	o.Val = n.V
	o.Present = n.Valid
	return nil
}
