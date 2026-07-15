package beego

import (
	"context"
	"os"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/polagonow/pola/repository"

	"beego-demo/db/models"
)

// TestMain wires beego's process-global database once. The repository
// constructor registers the Task model, so it runs before table sync.
func TestMain(m *testing.M) {
	if err := orm.RegisterDataBase("default", "sqlite3", "file:beegodemo?mode=memory&cache=shared"); err != nil {
		panic(err)
	}
	_ = NewTaskRepository(orm.NewOrm()) // registers the model with beego
	if err := orm.RunSyncdb("default", false, false); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// TestTaskRepository_CRUD verifies the beego-backed repository against a real
// sqlite database: create, read, update, paginate, delete.
func TestTaskRepository_CRUD(t *testing.T) {
	repo := NewTaskRepository(orm.NewOrm())
	ctx := context.Background()

	task := &models.Task{Title: "write docs"}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("expected auto-increment ID")
	}

	got, err := repo.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "write docs" || got.Done {
		t.Errorf("round-trip = %+v, want title kept and done=false", got)
	}

	got.Done = true
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if again, _ := repo.Get(ctx, task.ID); !again.Done {
		t.Error("expected Done=true after update")
	}

	list, err := repo.List(ctx, repository.ListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.Total < 1 {
		t.Errorf("Total = %d, want >= 1", list.Total)
	}

	if err := repo.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, task.ID); err == nil {
		t.Fatal("expected Get to error after Delete")
	}
}
