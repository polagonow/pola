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
	"time"
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

// LifecycleFields locates the framework-managed lifecycle columns of an entity
// T — CreatedAt/UpdatedAt timestamps and the soft-delete column — so the generic
// ORM adapters can stamp timestamps and soft-delete uniformly, with no ORM
// struct tags on the canonical model. It is computed once at construction.
type LifecycleFields struct {
	createdAt []int // index of a CreatedAt time.Time field, nil if absent
	updatedAt []int // index of an UpdatedAt time.Time field, nil if absent
	deletedAt []int // index of the soft-delete field, nil when soft-delete is off
	// DeletedAtColumn is the database column name of the soft-delete field.
	DeletedAtColumn string
}

// SoftDeletes reports whether entity T opts into soft deletion.
func (lf LifecycleFields) SoftDeletes() bool { return lf.deletedAt != nil }

var timeType = reflect.TypeOf(time.Time{})

// LifecycleFieldsOf inspects T for framework-managed lifecycle columns:
// CreatedAt/UpdatedAt (time.Time by conventional field name) and the soft-delete
// field (the field tagged pola:"soft_delete"). The model carries no ORM tags, so
// these conventions — not the ORM — drive lifecycle behavior.
func LifecycleFieldsOf[T any]() LifecycleFields {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	var lf LifecycleFields
	if t.Kind() != reflect.Struct {
		return lf
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if polaHasFlag(f.Tag.Get("pola"), "soft_delete") {
			lf.deletedAt = f.Index
			lf.DeletedAtColumn = snakeCase(f.Name)
			continue
		}
		if f.Type == timeType || (f.Type.Kind() == reflect.Pointer && f.Type.Elem() == timeType) {
			switch f.Name {
			case "CreatedAt":
				lf.createdAt = f.Index
			case "UpdatedAt":
				lf.updatedAt = f.Index
			}
		}
	}
	return lf
}

// polaHasFlag reports whether a semicolon-separated pola tag contains flag.
func polaHasFlag(tag, flag string) bool {
	for _, tok := range strings.Split(tag, ";") {
		if strings.TrimSpace(tok) == flag {
			return true
		}
	}
	return false
}

// StampCreate sets CreatedAt (only when zero) and UpdatedAt to now on the
// dereferenced entity value v. Fields the entity lacks are skipped.
func (lf LifecycleFields) StampCreate(v reflect.Value, now time.Time) {
	setTime(v, lf.createdAt, now, true)
	setTime(v, lf.updatedAt, now, false)
}

// StampUpdate sets UpdatedAt to now on the dereferenced entity value v.
func (lf LifecycleFields) StampUpdate(v reflect.Value, now time.Time) {
	setTime(v, lf.updatedAt, now, false)
}

// StampDelete sets the soft-delete column to now on the dereferenced entity
// value v. No-op when soft-delete is off.
func (lf LifecycleFields) StampDelete(v reflect.Value, now time.Time) {
	setTime(v, lf.deletedAt, now, false)
}

// IsSoftDeleted reports whether v's soft-delete column is set (the row is
// logically deleted). Always false when soft-delete is off.
func (lf LifecycleFields) IsSoftDeleted(v reflect.Value) bool {
	if lf.deletedAt == nil {
		return false
	}
	f := v.FieldByIndex(lf.deletedAt)
	if f.Kind() == reflect.Pointer {
		return !f.IsNil()
	}
	return !f.IsZero()
}

// setTime writes now into the time.Time (or *time.Time) field at index. When
// onlyIfZero is set, an already-populated value is left untouched.
func setTime(v reflect.Value, index []int, now time.Time, onlyIfZero bool) {
	if index == nil {
		return
	}
	f := v.FieldByIndex(index)
	if f.Kind() == reflect.Pointer {
		if onlyIfZero && !f.IsNil() {
			return
		}
		if f.IsNil() {
			f.Set(reflect.New(f.Type().Elem()))
		}
		f = f.Elem()
	} else if onlyIfZero && !f.IsZero() {
		return
	}
	f.Set(reflect.ValueOf(now))
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
