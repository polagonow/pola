// Package repository defines the framework-neutral persistence contract used
// by generated repositories, with generic ORM-backed implementations in the
// gorm and beego sub-packages. It mirrors webframework: one neutral contract,
// pluggable engines underneath. Ent repositories remain per-entity generated
// code because ent's typed codegen client (per-entity builders, per-field
// setters) has no generic surface to program against.
package repository

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// Repository is the standard CRUD-plus-pagination contract for entity T keyed
// by ID. Generated per-entity interfaces embed this so call sites keep a
// named, mockable, extensible interface (e.g. repositories.UserRepository).
type Repository[T any, ID comparable] interface {
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id ID) (*T, error)
	List(ctx context.Context, params ListParams) (*ListResult[*T], error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
}

// Settings carries implementation-agnostic construction options shared by the
// generic gorm and beego implementations.
type Settings[ID comparable] struct {
	// EntityName is the snake_case name used in wrapped error messages
	// ("get user by id: ..."). Empty means derive it from T's type name.
	EntityName string
	// NewID, when non-nil, is called by Create to assign a fresh ID to
	// entities whose ID field holds the zero value. The generator wires this
	// with uuid.NewString for string-keyed entities.
	NewID func() ID
}

// Option configures a generic repository implementation.
//
// Option is generic over ID only (not T) so WithNewID(uuid.NewString) infers
// its type parameter from the generator function at the call site.
type Option[ID comparable] func(*Settings[ID])

// WithNewID makes Create assign gen() to entities whose ID is the zero value.
func WithNewID[ID comparable](gen func() ID) Option[ID] {
	return func(s *Settings[ID]) { s.NewID = gen }
}

// WithEntityName overrides the entity name used in wrapped error messages.
// ID does not appear in the arguments, so it needs explicit instantiation:
// WithEntityName[string]("user").
func WithEntityName[ID comparable](name string) Option[ID] {
	return func(s *Settings[ID]) { s.EntityName = name }
}

// ApplySettings folds opts over zero Settings and derives EntityName from T
// when unset. Intended for use by implementation sub-packages.
func ApplySettings[T any, ID comparable](opts []Option[ID]) Settings[ID] {
	var s Settings[ID]
	for _, o := range opts {
		o(&s)
	}
	if s.EntityName == "" {
		s.EntityName = EntityNameOf[T]()
	}
	return s
}

// EntityNameOf returns T's type name in snake_case (SampleEntity ->
// "sample_entity"), used to label wrapped errors.
func EntityNameOf[T any]() string {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return snakeCase(t.Name())
}

// EnsureID assigns gen() to entity's ID field when that field currently holds
// the zero value. The entity must satisfy MustIDFieldIndex; generated
// entities always do.
func EnsureID[T any, ID comparable](entity *T, gen func() ID) {
	if entity == nil || gen == nil {
		return
	}
	f := reflect.ValueOf(entity).Elem().FieldByIndex(MustIDFieldIndex[T, ID]())
	var zero ID
	if f.Interface() == any(zero) {
		f.Set(reflect.ValueOf(gen()).Convert(f.Type()))
	}
}

// MustIDFieldIndex locates the index of T's primary-key struct field: the
// first exported field whose name equals "id" case-insensitively and whose
// type is convertible from ID. It panics with a descriptive message when T
// has no such field — that indicates a mis-declared entity, not a runtime
// condition. Implementations call it once at construction so bad entities
// fail fast.
func MustIDFieldIndex[T any, ID comparable]() []int {
	t := reflect.TypeFor[T]()
	idT := reflect.TypeFor[ID]()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.IsExported() && strings.EqualFold(sf.Name, "id") && idT.ConvertibleTo(sf.Type) {
			return sf.Index
		}
	}
	panic(fmt.Sprintf("repository: entity %s has no ID field of type %s", t.Name(), idT))
}

// snakeCase converts PascalCase to snake_case ("SampleEntity" ->
// "sample_entity"). Error-label cosmetics only; the generator has its own
// richer helper for filenames.
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
