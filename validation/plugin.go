package validation

import (
	"reflect"

	"github.com/polagonow/pola/core"
)

// Plugin installs the request validator used by the framework's Bind* methods
// (see core.MustBindValidate). Once installed, calling c.Bind(&entity) binds
// the request and then validates it against the struct's `validate:"..."` tags,
// writing a 422 on failure — so route handlers no longer repeat a manual
// validation.Validate step. The plugin is always on; validation.Validate
// remains available for callers that use the non-writing ShouldBind* variants.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "validation",
		Fn: func(r *core.Registry) {
			core.SetValidator(bindValidate)
		},
	}
}

// bindValidate validates struct values only. Non-struct binds (e.g. a scalar
// query parameter) are left untouched so they never yield a spurious 422.
func bindValidate(v any) error {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}
	return Validate(v)
}
