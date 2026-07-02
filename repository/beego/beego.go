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

	"github.com/beego/beego/v2/client/orm"

	"github.com/polagonow/pola/repository"
)

type repo[T any, ID comparable] struct {
	ormer    orm.Ormer
	settings repository.Settings[ID]
	idIndex  []int
}

// New returns a repository.Repository backed by o for entity T keyed by ID.
// It panics if T has no exported ID field of type ID (mis-declared entity).
func New[T any, ID comparable](o orm.Ormer, opts ...repository.Option[ID]) repository.Repository[T, ID] {
	idx := repository.MustIDFieldIndex[T, ID]()
	ensureRegistered(new(T))
	return &repo[T, ID]{ormer: o, settings: repository.ApplySettings[T, ID](opts), idIndex: idx}
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
	_, err := r.ormer.InsertWithCtx(ctx, entity)
	return err
}

func (r *repo[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	entity := r.withID(id)
	if err := r.ormer.ReadWithCtx(ctx, entity); err != nil {
		return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, err)
	}
	return entity, nil
}

func (r *repo[T, ID]) List(_ context.Context, params repository.ListParams) (*repository.ListResult[*T], error) {
	params = params.Normalize()

	qs := r.ormer.QueryTable(new(T))

	total, err := qs.Count()
	if err != nil {
		return nil, fmt.Errorf("count %s: %w", r.settings.EntityName, err)
	}

	var items []*T
	if _, err := qs.Limit(params.PerPage, params.Offset()).All(&items); err != nil {
		return nil, fmt.Errorf("list %s: %w", r.settings.EntityName, err)
	}

	return repository.NewListResult(items, int(total), params), nil
}

func (r *repo[T, ID]) Update(ctx context.Context, entity *T) error {
	_, err := r.ormer.UpdateWithCtx(ctx, entity)
	return err
}

func (r *repo[T, ID]) Delete(ctx context.Context, id ID) error {
	_, err := r.ormer.DeleteWithCtx(ctx, r.withID(id))
	return err
}
