package gorm

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gorm-demo/repositories"
)

// TestTaskRepository_CRUD is a wiring smoke test: it proves the entity
// auto-migrates and round-trips through the framework-backed repository.
// Behavioral coverage (pagination math, error wrapping, id generation) lives
// in the framework's repository conformance tests.
func TestTaskRepository_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&repositories.Task{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	repo := NewTaskRepository(db)
	ctx := context.Background()

	entity := &repositories.Task{}
	if err := repo.Create(ctx, entity); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected ID to be set by Create")
	}
	if _, err := repo.Get(ctx, entity.ID); err != nil {
		t.Fatalf("Get: %v", err)
	}
	result, err := repo.List(ctx, repositories.ListParams{Page: 1, PerPage: 10})
	if err != nil || result.Total < 1 {
		t.Fatalf("List: result=%+v err=%v", result, err)
	}
	if err := repo.Delete(ctx, entity.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, entity.ID); err == nil {
		t.Fatal("expected Get to error after Delete")
	}
}
