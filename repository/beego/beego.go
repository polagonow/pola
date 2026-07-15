// Package beego provides a generic Beego-ORM-backed implementation of
// repository.Repository.
//
// Unlike GORM, beego reads and deletes by primary key set ON the entity
// struct, so this implementation locates T's ID field once (by reflection) at
// construction. It also registers T with beego's model cache on first use:
// beego refuses to operate on unregistered models, and generated projects
// never registered repository entities. Registering relation-free models
// after bootstrap is permitted by beego's ModelCache.
package beego

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"github.com/polagonow/pola/repository"
)

type repo[T any, ID comparable] struct {
	ormer     orm.Ormer
	settings  repository.Settings[ID]
	idIndex   []int
	lifecycle repository.LifecycleFields
}

// New returns a repository.Repository backed by o for entity T keyed by ID.
// It panics if T has no exported ID field of type ID (mis-declared entity).
func New[T any, ID comparable](o orm.Ormer, opts ...repository.Option[ID]) repository.Repository[T, ID] {
	idx := repository.MustIDFieldIndex[T, ID]()
	ensureRegistered(new(T))
	return &repo[T, ID]{
		ormer:     o,
		settings:  repository.ApplySettings[T, ID](opts),
		idIndex:   idx,
		lifecycle: repository.LifecycleFieldsOf[T](),
	}
}

// baseQuery returns a QuerySeter for T with the soft-delete scope applied
// (deleted_at IS NULL) when T opts into soft deletion.
func (r *repo[T, ID]) baseQuery() orm.QuerySeter {
	qs := r.ormer.QueryTable(new(T))
	if r.lifecycle.SoftDeletes() {
		qs = qs.Filter(r.lifecycle.DeletedAtColumn+"__isnull", true)
	}
	return qs
}

// init selects beego's acronym-aware naming strategy. The canonical model
// carries no beego column tags, and beego's default naming maps "ID" → "i_d"
// and "AuthorID" → "author_i_d"; the acronym strategy maps them → "id" /
// "author_id", matching the migration and the other ORMs. It is a process-global
// setting and must be chosen before any RegisterModel, so it lives in init.
func init() {
	// Value of beego's models.SnakeAcronymNameStrategy (unexported); passed as a
	// literal because client/orm does not re-export the constant.
	orm.SetNameStrategy("snakeStringWithAcronym")
}

// registered tracks types this package has registered with beego, keyed by
// reflect.Type, so repeated constructor calls stay idempotent.
var registered sync.Map

// ensureRegistered registers the model with beego exactly once. If the model
// was already registered elsewhere (e.g. by a migration main), beego panics
// with a "repeat Register" error, which is recovered and ignored: the model
// being registered is precisely the state we want.
func ensureRegistered(model any) {
	t := reflect.TypeOf(model)
	if _, loaded := registered.LoadOrStore(t, struct{}{}); loaded {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok && strings.Contains(err.Error(), "repeat Register") {
				return
			}
			panic(r)
		}
	}()
	orm.RegisterModel(model)
}

// withID returns a new *T whose primary-key field is set to id.
func (r *repo[T, ID]) withID(id ID) *T {
	entity := new(T)
	f := reflect.ValueOf(entity).Elem().FieldByIndex(r.idIndex)
	f.Set(reflect.ValueOf(id).Convert(f.Type()))
	return entity
}

func (r *repo[T, ID]) Create(ctx context.Context, entity *T) error {
	if r.settings.NewID != nil {
		repository.EnsureID(entity, r.settings.NewID)
	}
	r.lifecycle.StampCreate(reflect.ValueOf(entity).Elem(), time.Now())
	_, err := r.ormer.InsertWithCtx(ctx, entity)
	return err
}

func (r *repo[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	entity := r.withID(id)
	if err := r.ormer.ReadWithCtx(ctx, entity); err != nil {
		return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, err)
	}
	// The neutral model carries no beego soft-delete tag, so filter here.
	if r.lifecycle.IsSoftDeleted(reflect.ValueOf(entity).Elem()) {
		return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, orm.ErrNoRows)
	}
	return entity, nil
}

func (r *repo[T, ID]) List(_ context.Context, params repository.ListParams) (*repository.ListResult[*T], error) {
	params = params.Normalize()

	total, err := r.baseQuery().Count()
	if err != nil {
		return nil, fmt.Errorf("count %s: %w", r.settings.EntityName, err)
	}

	var items []*T
	if _, err := r.baseQuery().Limit(params.PerPage, params.Offset()).All(&items); err != nil {
		return nil, fmt.Errorf("list %s: %w", r.settings.EntityName, err)
	}

	return repository.NewListResult(items, int(total), params), nil
}

func (r *repo[T, ID]) Update(ctx context.Context, entity *T) error {
	r.lifecycle.StampUpdate(reflect.ValueOf(entity).Elem(), time.Now())
	_, err := r.ormer.UpdateWithCtx(ctx, entity)
	return err
}

func (r *repo[T, ID]) Delete(ctx context.Context, id ID) error {
	// Soft delete: stamp the delete column on the row instead of removing it.
	if r.lifecycle.SoftDeletes() {
		entity := r.withID(id)
		r.lifecycle.StampDelete(reflect.ValueOf(entity).Elem(), time.Now())
		_, err := r.ormer.UpdateWithCtx(ctx, entity, r.lifecycle.DeletedAtColumn)
		return err
	}
	_, err := r.ormer.DeleteWithCtx(ctx, r.withID(id))
	return err
}
