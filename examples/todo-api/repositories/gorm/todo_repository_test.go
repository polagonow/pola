package gorm

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"todo-api/repositories"
)

// newTestRepo opens an in-memory SQLite database, auto-migrates the entity,
// and returns a fresh registry-constructed repository for use in a single test.
func newTestRepo(t *testing.T) (repositories.TodoRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&repositories.Todo{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	r := core.NewRegistry()
	core.ProvideValue[*gorm.DB](r, db)
	return NewTodoRepository(r), db
}

func TestTodoRepository_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	entity := &repositories.Todo{}
	if err := repo.Create(ctx, entity); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected ID to be set by Create")
	}

	got, err := repo.Get(ctx, entity.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != entity.ID {
		t.Fatalf("Get returned id %v want %v", got.ID, entity.ID)
	}
}

func TestTodoRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Create(ctx, &repositories.Todo{}); err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
	}

	result, err := repo.List(ctx, repositories.ListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Total < 3 {
		t.Fatalf("Total: got %d want >= 3", result.Total)
	}
}

func TestTodoRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	entity := &repositories.Todo{}
	if err := repo.Create(ctx, entity); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, entity.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, entity.ID); err == nil {
		t.Fatal("expected Get to error after Delete")
	}
}
