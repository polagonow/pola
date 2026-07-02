package gorm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
)

// TestWidget exercises the uint-keyed path; the multi-word name also covers
// EntityNameOf snake-casing in error labels.
type TestWidget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

// Session exercises the string/uuid-keyed path.
type Session struct {
	ID   string `gorm:"primaryKey"`
	Note string
}

func openDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func TestUintCRUD(t *testing.T) {
	db := openDB(t, &TestWidget{})
	repo := New[TestWidget, uint](db)
	ctx := context.Background()

	w := &TestWidget{Name: "alpha"}
	if err := repo.Create(ctx, w); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if w.ID == 0 {
		t.Fatal("expected auto-increment ID")
	}

	got, err := repo.Get(ctx, w.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "alpha" {
		t.Errorf("Name = %q, want alpha", got.Name)
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
}

func TestListPagination(t *testing.T) {
	db := openDB(t, &TestWidget{})
	repo := New[TestWidget, uint](db)
	ctx := context.Background()

	for _, n := range []string{"a", "b", "c"} {
		if err := repo.Create(ctx, &TestWidget{Name: n}); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}

	page1, err := repo.List(ctx, repository.ListParams{Page: 1, PerPage: 2})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if page1.Total != 3 || page1.TotalPages != 2 || len(page1.Items) != 2 {
		t.Errorf("page1 = total=%d pages=%d items=%d, want 3/2/2", page1.Total, page1.TotalPages, len(page1.Items))
	}

	page2, err := repo.List(ctx, repository.ListParams{Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("page2 items = %d, want 1", len(page2.Items))
	}

	// Zero params normalize to page 1 / DefaultPerPage.
	norm, err := repo.List(ctx, repository.ListParams{})
	if err != nil {
		t.Fatalf("List zero params: %v", err)
	}
	if norm.Page != 1 || norm.PerPage != repository.DefaultPerPage {
		t.Errorf("normalized = page=%d perPage=%d, want 1/%d", norm.Page, norm.PerPage, repository.DefaultPerPage)
	}
}

func TestStringIDWithNewID(t *testing.T) {
	db := openDB(t, &Session{})
	repo := New[Session, string](db, repository.WithNewID(uuid.NewString))
	ctx := context.Background()

	s := &Session{Note: "hi"}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == "" {
		t.Fatal("expected generated uuid ID")
	}
	if _, err := uuid.Parse(s.ID); err != nil {
		t.Errorf("ID %q is not a uuid: %v", s.ID, err)
	}

	pre := &Session{ID: "preset", Note: "keep"}
	if err := repo.Create(ctx, pre); err != nil {
		t.Fatalf("Create preset: %v", err)
	}
	if pre.ID != "preset" {
		t.Errorf("ID = %q, want preset kept", pre.ID)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get by uuid: %v", err)
	}
	if got.Note != "hi" {
		t.Errorf("Note = %q, want hi", got.Note)
	}
}

// TestStringIDInjection is the regression test for the raw-SQL hazard in the
// previous generated code (First(&entity, id) with a string id).
func TestStringIDInjection(t *testing.T) {
	db := openDB(t, &Session{})
	repo := New[Session, string](db, repository.WithNewID(uuid.NewString))
	ctx := context.Background()

	if err := repo.Create(ctx, &Session{Note: "target"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// With the old First(&e, id) form this string is spliced into the SQL and
	// matches every row; with the parameterized predicate it is just a
	// non-existent key.
	if _, err := repo.Get(ctx, "x' OR '1'='1"); err == nil {
		t.Fatal("expected not-found for hostile id, got a row (injection!)")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record-not-found, got: %v", err)
	}

	if err := repo.Delete(ctx, "x' OR '1'='1"); err != nil {
		t.Fatalf("Delete hostile id: %v", err)
	}
	res, err := repo.List(ctx, repository.ListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("Total = %d after hostile delete, want 1 (row must survive)", res.Total)
	}
}

func TestErrorLabels(t *testing.T) {
	db := openDB(t, &TestWidget{})
	repo := New[TestWidget, uint](db)

	_, err := repo.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing row")
	}
	if !strings.HasPrefix(err.Error(), "get test_widget by id:") {
		t.Errorf("error = %q, want prefix 'get test_widget by id:'", err)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected wrapped gorm.ErrRecordNotFound, got %v", err)
	}
}
