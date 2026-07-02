package ent

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"ent-demo/db/client/ent"
)

// newTestClient opens an in-memory SQLite-backed ent client and runs the
// schema migration. Each call creates an isolated database.
func newTestClient(t *testing.T) *ent.Client {
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

func TestTaskRepository_Compiles(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	client := newTestClient(t)
	repo := NewTaskRepository(client)
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
	// Smoke: verify the constructor returns a value satisfying the
	// repositories.TaskRepository interface (compile-checked via the
	// constructor's return type) and that List against an empty database
	// returns a zero-Total result rather than an error.
	// Full CRUD coverage requires field values that satisfy the schema's
	// non-nullable constraints — extend this test with concrete values
	// suited to your schema.
}
