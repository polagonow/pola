package polafile

import (
	"os"
	"reflect"
	"strconv"
)

// ApplyEnvOverrides reads POLA_* environment variables (identified by `env`
// struct tags) and applies them to pf, overriding any Polafile.hcl values.
// If pf is nil and at least one relevant env var is set, a new Polafile is
// returned. Fields are only touched when the corresponding env var is present
// in the environment; unset vars leave existing values intact.
func ApplyEnvOverrides(pf *Polafile) *Polafile {
	if pf == nil {
		pf = &Polafile{}
	}
	applyEnvToStruct(reflect.ValueOf(pf).Elem())
	return pf
}

func applyEnvToStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct {
			if structHasEnvVars(sf.Type.Elem()) {
				if field.IsNil() {
					field.Set(reflect.New(sf.Type.Elem()))
				}
				applyEnvToStruct(field.Elem())
			}
			continue
		}

		if sf.Type.Kind() == reflect.Struct {
			applyEnvToStruct(field)
			continue
		}

		envKey := sf.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envVal, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}

		setFieldFromEnv(field, sf, envVal)
	}
}

func setFieldFromEnv(field reflect.Value, sf reflect.StructField, val string) {
	switch sf.Type.Kind() {
	case reflect.String:
		field.SetString(val)
	case reflect.Bool:
		field.SetBool(val == "true" || val == "1")
	case reflect.Int:
		if n, err := strconv.Atoi(val); err == nil {
			field.SetInt(int64(n))
		}
	case reflect.Float64:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			field.SetFloat(f)
		}
	case reflect.Ptr:
		if sf.Type.Elem().Kind() == reflect.Bool {
			b := val == "true" || val == "1"
			field.Set(reflect.ValueOf(&b))
		}
	}
}

// structHasEnvVars reports whether any field in t carries an `env` tag whose
// corresponding environment variable is currently set.
func structHasEnvVars(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if envKey := t.Field(i).Tag.Get("env"); envKey != "" {
			if _, ok := os.LookupEnv(envKey); ok {
				return true
			}
		}
	}
	return false
}
