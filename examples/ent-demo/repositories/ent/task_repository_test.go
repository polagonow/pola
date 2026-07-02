package ent

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/polagonow/pola/repository"

	"ent-demo/db/client/ent"
	"ent-demo/repositories"
)

// newTaskTestClient opens an in-memory SQLite-backed ent client and runs
// the schema migration. Each call creates an isolated database.
func newTaskTestClient(t *testing.T) *ent.Client {
	t.Helper()
	client, err := ent.Open("sqlite3", "file:ent_task_test?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("schema create: %v", err)
	}
	return client
}

// TestTaskRepository_CRUD is a wiring smoke test: it proves the entity
// round-trips through the framework-backed repository against the generated
// ent client. Behavioral coverage (pagination math, error wrapping, field
// dispatch) lives in the framework's repository conformance tests.
func TestTaskRepository_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	repo := NewTaskRepository(newTaskTestClient(t))
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
	result, err := repo.List(ctx, repository.ListParams{Page: 1, PerPage: 10})
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
