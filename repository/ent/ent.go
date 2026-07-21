// Package ent provides a generic ent-backed implementation of
// repository.Repository.
//
// Ent's generated client is typed per entity (client.Task.Create() with
// per-field setters), and the framework cannot import an app's generated
// client package. This implementation therefore binds to the client by
// reflection — once, at construction, with fail-fast validation — and drives
// Create/Update through ent's own dynamic ent.Mutation interface, whose
// generated SetField(name, value) validates field names and types at runtime.
// Everything else (Query/Get/UpdateOneID/DeleteOneID and the fluent
// Offset/Limit/Count/All builders) is ent's stable, structurally uniform
// codegen surface.
package ent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	entgo "entgo.io/ent"
	entsql "entgo.io/ent/dialect/sql"

	"github.com/polagonow/pola/repository"
)

type repo[T any, ID comparable] struct {
	sub       reflect.Value // e.g. client.Task (*TaskClient)
	settings  repository.Settings[ID]
	idIndex   []int        // T's ID field
	entIDTyp  reflect.Type // ent's ID type (int or string), from Get's 2nd param
	fields    []fieldPlan  // exported non-ID fields of T -> mutation field names
	lifecycle repository.LifecycleFields
}

// fieldPlan maps one exported struct field of T to its ent schema field name.
type fieldPlan struct {
	index []int
	name  string
}

// New returns a repository.Repository backed by the app's generated ent
// client for entity T keyed by ID. client is the *ent.Client passed as any
// (the framework cannot name the app's generated type). It panics at
// construction — DI provide time — when the client has no sub-client named
// after T or the sub-client is missing the expected codegen methods.
//
//	repo := entrepo.New[repositories.Task, uint](client)
//	repo := entrepo.New[repositories.Session, string](client, repository.WithNewID(uuid.NewString))
func New[T any, ID comparable](client any, opts ...repository.Option[ID]) repository.Repository[T, ID] {
	t := reflect.TypeFor[T]()
	cv := reflect.ValueOf(client)
	if !cv.IsValid() || cv.Kind() != reflect.Pointer || cv.IsNil() || cv.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("repository/ent: client must be a non-nil pointer to the generated ent client, got %T", client))
	}
	sub := cv.Elem().FieldByName(t.Name())
	if !sub.IsValid() {
		panic(fmt.Sprintf("repository/ent: ent client %T has no %s sub-client (entity name must match the ent schema)", client, t.Name()))
	}
	for _, m := range []string{"Create", "Query", "Get", "UpdateOneID", "DeleteOneID"} {
		if !sub.MethodByName(m).IsValid() {
			panic(fmt.Sprintf("repository/ent: %s sub-client has no %s method", t.Name(), m))
		}
	}
	get := sub.MethodByName("Get").Type()
	if get.NumIn() != 2 || get.NumOut() != 2 {
		panic(fmt.Sprintf("repository/ent: unexpected %s.Get signature %s", t.Name(), get))
	}
	return &repo[T, ID]{
		sub:       sub,
		settings:  repository.ApplySettings[T, ID](opts),
		idIndex:   repository.MustIDFieldIndex[T, ID](),
		entIDTyp:  get.In(1),
		fields:    buildFieldPlan(t),
		lifecycle: repository.LifecycleFieldsOf[T](),
	}
}

// scopedQuery returns a fresh query with the soft-delete scope applied
// (deleted_at IS NULL) when entity T opts into soft deletion.
func (r *repo[T, ID]) scopedQuery() reflect.Value {
	q := r.sub.MethodByName("Query").Call(nil)[0]
	if r.lifecycle.SoftDeletes() {
		q = whereField(q, entsql.FieldIsNull(r.lifecycle.DeletedAtColumn))
	}
	return q
}

// whereField applies a dialect/sql field predicate (a func(*sql.Selector)) to an
// ent query builder via reflection, converting it to the query's generated
// predicate.T type. This lets the framework filter without importing the app's
// generated predicate package.
func whereField(q reflect.Value, fn any) reflect.Value {
	where := q.MethodByName("Where")
	predType := where.Type().In(0).Elem() // predicate.T (named func(*sql.Selector))
	pred := reflect.ValueOf(fn).Convert(predType)
	return where.Call([]reflect.Value{pred})[0]
}

// buildFieldPlan maps T's exported non-ID fields to ent schema field names:
// the first json-tag segment, falling back to the snake-cased Go name. Both
// the repositories struct and the ent schema derive from the same CLI
// definition, so the names line up.
func buildFieldPlan(t reflect.Type) []fieldPlan {
	var plan []fieldPlan
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() || strings.EqualFold(sf.Name, "id") {
			continue
		}
		name := sf.Name
		if tag, ok := sf.Tag.Lookup("json"); ok {
			if comma := strings.IndexByte(tag, ','); comma >= 0 {
				tag = tag[:comma]
			}
			if tag == "-" {
				continue
			}
			if tag != "" {
				name = tag
			} else {
				name = snakeCase(sf.Name)
			}
		} else {
			name = snakeCase(sf.Name)
		}
		plan = append(plan, fieldPlan{index: sf.Index, name: name})
	}
	return plan
}

func (r *repo[T, ID]) Create(ctx context.Context, entity *T) error {
	if r.settings.NewID != nil {
		repository.EnsureID(entity, r.settings.NewID)
	}
	r.lifecycle.StampCreate(reflect.ValueOf(entity).Elem(), r.settings.Clock())
	builder := r.sub.MethodByName("Create").Call(nil)[0]
	// Honor a caller-supplied ID when the schema allows it (string-ID schemas
	// generate a SetID; auto-increment int schemas do not).
	if setID := builder.MethodByName("SetID"); setID.IsValid() {
		idVal := reflect.ValueOf(entity).Elem().FieldByIndex(r.idIndex)
		if !idVal.IsZero() {
			builder = setID.Call([]reflect.Value{idVal.Convert(r.entIDTyp)})[0]
		}
	}
	if err := r.applyMutation(builder, entity); err != nil {
		return err
	}
	saved, err := callSave(ctx, builder)
	if err != nil {
		return err
	}
	copyFields(saved, reflect.ValueOf(entity).Elem())
	return nil
}

func (r *repo[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	entID := reflect.ValueOf(id).Convert(r.entIDTyp)

	// When soft-delete is on, fetch through Query so a soft-deleted row is
	// excluded (deleted_at IS NULL) and Only returns ent's NotFound.
	if r.lifecycle.SoftDeletes() {
		q := whereField(r.scopedQuery(), entsql.FieldEQ("id", entID.Interface()))
		out := q.MethodByName("Only").Call([]reflect.Value{reflect.ValueOf(ctx)})
		if err := asError(out[1]); err != nil {
			return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, mapNotFound(err))
		}
		entity := new(T)
		copyFields(out[0], reflect.ValueOf(entity).Elem())
		return entity, nil
	}

	out := r.sub.MethodByName("Get").Call([]reflect.Value{
		reflect.ValueOf(ctx), entID,
	})
	if err := asError(out[1]); err != nil {
		return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, mapNotFound(err))
	}
	entity := new(T)
	copyFields(out[0], reflect.ValueOf(entity).Elem())
	return entity, nil
}

func (r *repo[T, ID]) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*T], error) {
	params = params.Normalize()

	count := r.scopedQuery().MethodByName("Count").
		Call([]reflect.Value{reflect.ValueOf(ctx)})
	if err := asError(count[1]); err != nil {
		return nil, fmt.Errorf("count %s: %w", r.settings.EntityName, err)
	}
	total := int(count[0].Int())

	q := r.scopedQuery()
	q = q.MethodByName("Offset").Call([]reflect.Value{reflect.ValueOf(params.Offset())})[0]
	q = q.MethodByName("Limit").Call([]reflect.Value{reflect.ValueOf(params.PerPage)})[0]
	all := q.MethodByName("All").Call([]reflect.Value{reflect.ValueOf(ctx)})
	if err := asError(all[1]); err != nil {
		return nil, fmt.Errorf("list %s: %w", r.settings.EntityName, err)
	}

	rows := all[0]
	items := make([]*T, rows.Len())
	for i := range items {
		items[i] = new(T)
		copyFields(rows.Index(i), reflect.ValueOf(items[i]).Elem())
	}
	return repository.NewListResult(items, total, params), nil
}

func (r *repo[T, ID]) Update(ctx context.Context, entity *T) error {
	r.lifecycle.StampUpdate(reflect.ValueOf(entity).Elem(), r.settings.Clock())
	idVal := reflect.ValueOf(entity).Elem().FieldByIndex(r.idIndex)
	builder := r.sub.MethodByName("UpdateOneID").
		Call([]reflect.Value{idVal.Convert(r.entIDTyp)})[0]
	if err := r.applyMutation(builder, entity); err != nil {
		return err
	}
	_, err := callSave(ctx, builder)
	return err
}

func (r *repo[T, ID]) Delete(ctx context.Context, id ID) error {
	// Soft delete: stamp deleted_at instead of removing the row.
	if r.lifecycle.SoftDeletes() {
		builder := r.sub.MethodByName("UpdateOneID").
			Call([]reflect.Value{reflect.ValueOf(id).Convert(r.entIDTyp)})[0]
		mut, ok := builder.MethodByName("Mutation").Call(nil)[0].Interface().(entgo.Mutation)
		if !ok {
			return fmt.Errorf("delete %s: builder mutation does not implement ent.Mutation", r.settings.EntityName)
		}
		if err := mut.SetField(r.lifecycle.DeletedAtColumn, r.settings.Clock()); err != nil {
			return fmt.Errorf("delete %s: %w", r.settings.EntityName, err)
		}
		if _, err := callSave(ctx, builder); err != nil {
			return fmt.Errorf("delete %s: %w", r.settings.EntityName, err)
		}
		return nil
	}
	builder := r.sub.MethodByName("DeleteOneID").
		Call([]reflect.Value{reflect.ValueOf(id).Convert(r.entIDTyp)})[0]
	out := builder.MethodByName("Exec").Call([]reflect.Value{reflect.ValueOf(ctx)})
	return asError(out[0])
}

// applyMutation copies entity's planned fields into the builder's mutation
// via ent's dynamic SetField, which validates names and types at runtime.
func (r *repo[T, ID]) applyMutation(builder reflect.Value, entity *T) error {
	mut, ok := builder.MethodByName("Mutation").Call(nil)[0].Interface().(entgo.Mutation)
	if !ok {
		return fmt.Errorf("%s builder mutation does not implement ent.Mutation", r.settings.EntityName)
	}
	ev := reflect.ValueOf(entity).Elem()
	for _, f := range r.fields {
		// The soft-delete column is written only by Delete, never here.
		if r.lifecycle.SoftDeletes() && f.name == r.lifecycle.DeletedAtColumn {
			continue
		}
		v := ev.FieldByIndex(f.index)
		// Optional/nullable fields are pointers: skip when nil (leave the column
		// null), otherwise dereference to the value ent's SetField expects.
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if err := mut.SetField(f.name, normalize(v)); err != nil {
			return fmt.Errorf("set %s.%s: %w", r.settings.EntityName, f.name, err)
		}
	}
	return nil
}

// normalize adapts repository field values to ent's expected Go types. The
// only divergence the generators produce is unsigned ints (e.g. StorageBlob
// FK fields are uint) where ent int fields expect int.
func normalize(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint())
	default:
		return v.Interface()
	}
}

// callSave invokes builder.Save(ctx), returning the saved ent entity value.
func callSave(ctx context.Context, builder reflect.Value) (reflect.Value, error) {
	out := builder.MethodByName("Save").Call([]reflect.Value{reflect.ValueOf(ctx)})
	return out[0], asError(out[1])
}

// copyFields copies exported, name-matched, convertible fields from the ent
// entity (pointer to generated struct) onto dst (the repositories struct).
// This maps query results back and fills auto-generated IDs after Create.
func copyFields(src reflect.Value, dst reflect.Value) {
	for src.Kind() == reflect.Pointer {
		src = src.Elem()
	}
	if !src.IsValid() || src.Kind() != reflect.Struct {
		return
	}
	dt := dst.Type()
	for i := 0; i < dt.NumField(); i++ {
		df := dt.Field(i)
		if !df.IsExported() {
			continue
		}
		sv := src.FieldByName(df.Name)
		if !sv.IsValid() || !sv.Type().ConvertibleTo(df.Type) {
			continue
		}
		dst.Field(i).Set(sv.Convert(df.Type))
	}
}

// asError unwraps a reflected error return value.
func asError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return v.Interface().(error)
}

// mapNotFound converts ent's generated *NotFoundError to the ORM-neutral
// repository.ErrNotFound, leaving other errors unchanged. The framework cannot
// import the app's ent package to call its IsNotFound, so it matches the
// concrete type name ent gives its not-found error ("NotFoundError") in every
// generated package.
func mapNotFound(err error) error {
	for e := err; e != nil; e = errors.Unwrap(e) {
		t := reflect.TypeOf(e)
		if t != nil && t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t != nil && t.Name() == "NotFoundError" {
			return repository.ErrNotFound
		}
	}
	return err
}

// snakeCase converts PascalCase to snake_case ("AvatarID" -> "avatar_id"),
// matching the generators' naming for untagged fields.
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && (s[i-1] < 'A' || s[i-1] > 'Z') {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
