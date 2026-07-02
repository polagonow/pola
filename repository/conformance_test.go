package repository_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	beegoorm "github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	beegorepo "github.com/polagonow/pola/repository/beego"
	gormrepo "github.com/polagonow/pola/repository/gorm"
)

// Widget is the shared conformance entity. It carries both ORMs' tags — the
// same dual role a generated repositories.X entity plays.
type Widget struct {
	ID   uint   `gorm:"primaryKey" orm:"column(id);auto;pk"`
	Name string `orm:"column(name)"`
}

// TableName pins one table name for both ORMs.
func (Widget) TableName() string { return "widgets" }

func newGormWidgetRepo(t *testing.T) repository.Repository[Widget, uint] {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Widget{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return gormrepo.New[Widget, uint](db)
}

// beegoSetup wires beego's process-global default database once; the model is
// registered by the repository constructor itself, before table sync.
var beegoSetup sync.Once

func newBeegoWidgetRepo(t *testing.T) repository.Repository[Widget, uint] {
	t.Helper()
	beegoSetup.Do(func() {
		if err := beegoorm.RegisterDataBase("default", "sqlite3", "file:conformance?mode=memory&cache=shared"); err != nil {
			panic(err)
		}
		_ = beegorepo.New[Widget, uint](beegoorm.NewOrm()) // registers the model
		if err := beegoorm.RunSyncdb("default", false, false); err != nil {
			panic(err)
		}
	})
	return beegorepo.New[Widget, uint](beegoorm.NewOrm())
}

// TestConformance runs the same observable-behavior suite against every
// generic ORM implementation — the persistence twin of
// webframework/conformance_test.go. Each adapter starts on an empty store and
// the suite is ordered so row counts stay deterministic.
func TestConformance(t *testing.T) {
	factories := map[string]func(t *testing.T) repository.Repository[Widget, uint]{
		"gorm":  newGormWidgetRepo,
		"beego": newBeegoWidgetRepo,
	}

	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			repo := factory(t)
			ctx := context.Background()

			t.Run("crud", func(t *testing.T) {
				w := &Widget{Name: "alpha"}
				if err := repo.Create(ctx, w); err != nil {
					t.Fatalf("Create: %v", err)
				}
				if w.ID == 0 {
					t.Fatal("expected assigned ID")
				}
				got, err := repo.Get(ctx, w.ID)
				if err != nil {
					t.Fatalf("Get: %v", err)
				}
				if got.ID != w.ID || got.Name != "alpha" {
					t.Errorf("round-trip = %+v, want %+v", got, w)
				}
				got.Name = "beta"
				if err := repo.Update(ctx, got); err != nil {
					t.Fatalf("Update: %v", err)
				}
				if again, _ := repo.Get(ctx, w.ID); again.Name != "beta" {
					t.Errorf("Name after update = %q, want beta", again.Name)
				}
				if err := repo.Delete(ctx, w.ID); err != nil {
					t.Fatalf("Delete: %v", err)
				}
				if _, err := repo.Get(ctx, w.ID); err == nil {
					t.Fatal("expected Get to error after Delete")
				}
			})

			t.Run("pagination", func(t *testing.T) {
				for _, n := range []string{"a", "b", "c"} {
					if err := repo.Create(ctx, &Widget{Name: n}); err != nil {
						t.Fatalf("Create %s: %v", n, err)
					}
				}
				page1, err := repo.List(ctx, repository.ListParams{Page: 1, PerPage: 2})
				if err != nil {
					t.Fatalf("List: %v", err)
				}
				if page1.Total != 3 || page1.TotalPages != 2 || len(page1.Items) != 2 || page1.Page != 1 || page1.PerPage != 2 {
					t.Errorf("page1 = %+v, want total=3 pages=2 items=2", page1)
				}
				page2, err := repo.List(ctx, repository.ListParams{Page: 2, PerPage: 2})
				if err != nil {
					t.Fatalf("List page 2: %v", err)
				}
				if len(page2.Items) != 1 {
					t.Errorf("page2 items = %d, want 1", len(page2.Items))
				}
				norm, err := repo.List(ctx, repository.ListParams{})
				if err != nil {
					t.Fatalf("List zero params: %v", err)
				}
				if norm.Page != 1 || norm.PerPage != repository.DefaultPerPage {
					t.Errorf("normalized = page=%d perPage=%d, want 1/%d", norm.Page, norm.PerPage, repository.DefaultPerPage)
				}
			})

			t.Run("error_label", func(t *testing.T) {
				_, err := repo.Get(ctx, 99999)
				if err == nil {
					t.Fatal("expected error for missing row")
				}
				// Both adapters share the wrapped-message format; the wrapped
				// driver error differs (gorm.ErrRecordNotFound vs orm.ErrNoRows)
				// until a cross-ORM repository.ErrNotFound exists.
				if !strings.HasPrefix(err.Error(), "get widget by id:") {
					t.Errorf("error = %q, want prefix 'get widget by id:'", err)
				}
			})
		})
	}
}
