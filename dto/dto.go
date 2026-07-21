// Package dto provides the presentation-layer data-transfer types that sit
// between a pola app's HTTP/RSC boundary and its domain models, plus the helpers
// that map between them. It borrows the shape proven by Goyave's typeutil: keep
// database models in the data layer, expose DTOs to clients, and move data
// across the boundary with a small, explicit set of converters.
//
//   - [Convert]/[MustConvert] turn any value (a model, a map, another DTO) into a
//     typed DTO, keeping only the fields the DTO declares — so a response DTO
//     naturally strips columns it doesn't list, and unexpected client input is
//     dropped rather than bound.
//   - [Copy] writes a DTO's fields onto a model for persistence, honoring
//     [Optional] so partial-update (PATCH) requests touch only supplied fields.
//   - [Optional] distinguishes an absent field from one explicitly set to its
//     zero value, which is what makes correct PATCH semantics possible.
//
// The mapping is deliberately dependency-free (reflection + encoding/json),
// matching pola's Go-native, minimal-dependency philosophy.
package dto

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Convert marshals src to JSON and unmarshals it into T. The JSON round-trip is
// what filters fields: only keys that T declares are retained, so converting a
// model to a response DTO drops columns the DTO omits, and converting raw client
// input to a DTO drops unexpected extras. Optional fields on T decode as present
// only when the corresponding key exists in src.
func Convert[T any](src any) (T, error) {
	var out T
	b, err := json.Marshal(src)
	if err != nil {
		return out, fmt.Errorf("dto: marshal source: %w", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("dto: unmarshal into %T: %w", out, err)
	}
	return out, nil
}

// MustConvert is [Convert] that panics on error. Use it where conversion cannot
// legitimately fail — e.g. converting an already-validated model to its response
// DTO — so handlers stay free of unreachable error branches.
func MustConvert[T any](src any) T {
	out, err := Convert[T](src)
	if err != nil {
		panic(err)
	}
	return out
}

// presenceField is the protocol implemented by [Optional]. [Copy] uses it to
// read a wrapped field's presence and payload without knowing its type
// parameter.
type presenceField interface {
	IsPresent() bool
	value() any
}

// Copy writes the fields of DTO src onto the model pointed to by dst, matching
// by exported field name, and returns dst for chaining. It is the DTO→model
// direction used before persisting:
//
//   - a field wrapped in [Optional] is written only when present (so an absent
//     PATCH field is left untouched, while an explicit zero IS written);
//   - a plain field is written only when it is non-zero (so a partially-filled
//     create DTO doesn't clobber model defaults);
//   - fields with no name match on the model, or with incompatible types, are
//     skipped.
//
// This mirrors Goyave's typeutil.Copy but without the copier dependency. Fetch
// the current model first, Copy the update DTO onto it, then save — that read-
// modify-write avoids the temporal inconsistency of building a fresh model from
// a partial DTO.
func Copy[M any, D any](dst *M, src D) *M {
	if dst == nil {
		return dst
	}
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src)
	for sv.Kind() == reflect.Pointer {
		if sv.IsNil() {
			return dst
		}
		sv = sv.Elem()
	}
	if dv.Kind() != reflect.Struct || sv.Kind() != reflect.Struct {
		return dst
	}

	st := sv.Type()
	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		fv := sv.Field(i)

		var payload any
		if p, ok := fv.Interface().(presenceField); ok {
			if !p.IsPresent() {
				continue // absent optional: leave the model field as-is
			}
			payload = p.value()
		} else {
			if fv.IsZero() {
				continue // zero plain field: don't overwrite model defaults
			}
			payload = fv.Interface()
		}

		target := dv.FieldByName(sf.Name)
		if !target.IsValid() || !target.CanSet() {
			continue
		}
		assign(target, payload)
	}
	return dst
}

// assign stores payload into target, converting between compatible types
// (e.g. a DTO's int64 into a model's int). Incompatible pairings are skipped so
// a mismatched field never panics the mapper.
func assign(target reflect.Value, payload any) {
	pv := reflect.ValueOf(payload)
	if !pv.IsValid() {
		return
	}
	tt := target.Type()
	switch {
	case pv.Type().AssignableTo(tt):
		target.Set(pv)
	case pv.Type().ConvertibleTo(tt):
		target.Set(pv.Convert(tt))
	}
}
