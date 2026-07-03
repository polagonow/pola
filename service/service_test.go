package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/service"
)

type widget struct {
	ID   uint
	Name string
}

// fakeRepo records the last call to each method and returns preset values,
// so the tests can prove the generic service delegates faithfully.
type fakeRepo struct {
	created, updated *widget
	gotID, deletedID uint
	listParams       repository.ListParams

	getResult  *widget
	listResult *repository.ListResult[*widget]
	err        error
}

func (f *fakeRepo) Create(_ context.Context, e *widget) error { f.created = e; return f.err }
func (f *fakeRepo) Get(_ context.Context, id uint) (*widget, error) {
	f.gotID = id
	return f.getResult, f.err
}
func (f *fakeRepo) List(_ context.Context, p repository.ListParams) (*repository.ListResult[*widget], error) {
	f.listParams = p
	return f.listResult, f.err
}
func (f *fakeRepo) Update(_ context.Context, e *widget) error { f.updated = e; return f.err }
func (f *fakeRepo) Delete(_ context.Context, id uint) error   { f.deletedID = id; return f.err }

var _ repository.Repository[widget, uint] = (*fakeRepo)(nil)

func TestServiceDelegates(t *testing.T) {
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := service.New[widget, uint](repo)
		w := &widget{Name: "a"}
		if err := svc.Create(ctx, w); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if repo.created != w {
			t.Errorf("repo.Create got %v, want %v", repo.created, w)
		}
	})

	t.Run("get", func(t *testing.T) {
		want := &widget{ID: 7, Name: "g"}
		repo := &fakeRepo{getResult: want}
		svc := service.New[widget, uint](repo)
		got, err := svc.Get(ctx, 7)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got != want || repo.gotID != 7 {
			t.Errorf("Get got %v (id %d), want %v (id 7)", got, repo.gotID, want)
		}
	})

	t.Run("list", func(t *testing.T) {
		want := &repository.ListResult[*widget]{Total: 3, Page: 1, PerPage: 2}
		repo := &fakeRepo{listResult: want}
		svc := service.New[widget, uint](repo)
		got, err := svc.List(ctx, repository.ListParams{Page: 1, PerPage: 2})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got != want || repo.listParams.PerPage != 2 {
			t.Errorf("List got %v (params %+v), want %v", got, repo.listParams, want)
		}
	})

	t.Run("update", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := service.New[widget, uint](repo)
		w := &widget{ID: 1, Name: "u"}
		if err := svc.Update(ctx, w); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if repo.updated != w {
			t.Errorf("repo.Update got %v, want %v", repo.updated, w)
		}
	})

	t.Run("delete", func(t *testing.T) {
		repo := &fakeRepo{}
		svc := service.New[widget, uint](repo)
		if err := svc.Delete(ctx, 5); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if repo.deletedID != 5 {
			t.Errorf("repo.Delete got id %d, want 5", repo.deletedID)
		}
	})
}

func TestServicePropagatesError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("boom")
	svc := service.New[widget, uint](&fakeRepo{err: sentinel})

	if err := svc.Create(ctx, &widget{}); !errors.Is(err, sentinel) {
		t.Errorf("Create err = %v, want %v", err, sentinel)
	}
	if _, err := svc.Get(ctx, 1); !errors.Is(err, sentinel) {
		t.Errorf("Get err = %v, want %v", err, sentinel)
	}
	if _, err := svc.List(ctx, repository.ListParams{}); !errors.Is(err, sentinel) {
		t.Errorf("List err = %v, want %v", err, sentinel)
	}
	if err := svc.Update(ctx, &widget{}); !errors.Is(err, sentinel) {
		t.Errorf("Update err = %v, want %v", err, sentinel)
	}
	if err := svc.Delete(ctx, 1); !errors.Is(err, sentinel) {
		t.Errorf("Delete err = %v, want %v", err, sentinel)
	}
}
