// Package gorm provides a generic GORM-backed implementation of
// repository.Repository. One constructor call in generated code replaces the
// previous ~70-line per-entity implementation.
package gorm

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/polagonow/pola/repository"
)

type repo[T any, ID comparable] struct {
	db        *gorm.DB
	settings  repository.Settings[ID]
	lifecycle repository.LifecycleFields
}

// New returns a repository.Repository backed by db for entity T keyed by ID.
//
//	repo := gorm.New[models.User, uint](db)
//	repo := gorm.New[models.Session, string](db, repository.WithNewID(uuid.NewString))
func New[T any, ID comparable](db *gorm.DB, opts ...repository.Option[ID]) repository.Repository[T, ID] {
	s := repository.ApplySettings[T, ID](opts)
	if s.NewID != nil {
		// Fail fast on entities without a usable ID field.
		repository.MustIDFieldIndex[T, ID]()
	}
	return &repo[T, ID]{db: db, settings: s, lifecycle: repository.LifecycleFieldsOf[T]()}
}

func (r *repo[T, ID]) Create(ctx context.Context, entity *T) error {
	if r.settings.NewID != nil {
		repository.EnsureID(entity, r.settings.NewID)
	}
	r.lifecycle.StampCreate(reflect.ValueOf(entity).Elem(), time.Now())
	return r.db.WithContext(ctx).Create(entity).Error
}

// query applies the soft-delete scope (deleted_at IS NULL) when entity T opts
// into soft deletion, so soft-deleted rows are invisible to reads. The model
// carries no gorm.DeletedAt, so this scope — not gorm — enforces it.
func (r *repo[T, ID]) query(ctx context.Context) *gorm.DB {
	q := r.db.WithContext(ctx)
	if r.lifecycle.SoftDeletes() {
		q = q.Where(r.lifecycle.DeletedAtColumn + " IS NULL")
	}
	return q
}

// byID builds a primary-key predicate explicitly. First(&entity, id) treats a
// non-numeric string condition as a raw SQL fragment — a SQL injection hazard
// for uuid-keyed entities. clause.Eq on clause.PrimaryColumn produces the same
// "WHERE id = ?" for every key type.
func (r *repo[T, ID]) byID(ctx context.Context, id ID) *gorm.DB {
	return r.query(ctx).Where(clause.Eq{Column: clause.PrimaryColumn, Value: id})
}

func (r *repo[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	var entity T
	if err := r.byID(ctx, id).First(&entity).Error; err != nil {
		return nil, fmt.Errorf("get %s by id: %w", r.settings.EntityName, err)
	}
	return &entity, nil
}

func (r *repo[T, ID]) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*T], error) {
	params = params.Normalize()

	var total int64
	if err := r.query(ctx).Model(new(T)).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count %s: %w", r.settings.EntityName, err)
	}

	var items []*T
	if err := r.query(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list %s: %w", r.settings.EntityName, err)
	}

	return repository.NewListResult(items, int(total), params), nil
}

func (r *repo[T, ID]) Update(ctx context.Context, entity *T) error {
	r.lifecycle.StampUpdate(reflect.ValueOf(entity).Elem(), time.Now())
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *repo[T, ID]) Delete(ctx context.Context, id ID) error {
	if r.lifecycle.SoftDeletes() {
		res := r.byID(ctx, id).Model(new(T)).Update(r.lifecycle.DeletedAtColumn, time.Now())
		if res.Error != nil {
			return fmt.Errorf("delete %s: %w", r.settings.EntityName, res.Error)
		}
		return nil
	}
	return r.byID(ctx, id).Delete(new(T)).Error
}
