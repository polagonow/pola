package ent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/polagonow/pola/repository"
	entfix "github.com/polagonow/pola/repository/ent/enttest/ent"
)

// openClient returns an isolated in-memory fixture client with the schema
// migrated. name must be unique per test so databases don't collide.
func openClient(t *testing.T, name string) *entfix.Client {
	t.Helper()
	client, err := entfix.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", name))
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("schema create: %v", err)
	}
	return client
}

func TestUintCRUD(t *testing.T) {
	type Widget struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	client := openClient(t, "entrepo_crud")
	repo := New[Widget, uint](client)
	ctx := context.Background()

	w := &Widget{Name: "alpha"}
	if err := repo.Create(ctx, w); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// The old generated template discarded the auto ID; the generic impl must
	// copy it back (parity with gorm/beego).
	if w.ID == 0 {
		t.Fatal("expected auto-generated ID copied back after Create")
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
	if err := repo.Delete(ctx, w.ID); err == nil {
		t.Fatal("expected Delete of missing row to error (ent NotFoundError)")
	}
}

func TestTypedFields(t *testing.T) {
	type TypedWidget struct {
		ID    uint    `json:"id"`
		Title string  `json:"title"`
		Count int     `json:"count"`
		Rate  float64 `json:"rate"`
		Done  bool    `json:"done"`
	}
	client := openClient(t, "entrepo_typed")
	repo := New[TypedWidget, uint](client)
	ctx := context.Background()

	in := &TypedWidget{Title: "t", Count: -3, Rate: 2.5, Done: true}
	if err := repo.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.Get(ctx, in.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "t" || got.Count != -3 || got.Rate != 2.5 || !got.Done {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestListPagination(t *testing.T) {
	type Widget struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	client := openClient(t, "entrepo_list")
	repo := New[Widget, uint](client)
	ctx := context.Background()

	for _, n := range []string{"a", "b", "c"} {
		if err := repo.Create(ctx, &Widget{Name: n}); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}

	page1, err := repo.List(ctx, repository.ListParams{Page: 1, PerPage: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
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
	norm, err := repo.List(ctx, repository.ListParams{})
	if err != nil {
		t.Fatalf("List zero params: %v", err)
	}
	if norm.Page != 1 || norm.PerPage != repository.DefaultPerPage {
		t.Errorf("normalized = page=%d perPage=%d, want 1/%d", norm.Page, norm.PerPage, repository.DefaultPerPage)
	}
}

func TestStringIDSession(t *testing.T) {
	type Session struct {
		ID   string `json:"id"`
		Note string `json:"note"`
	}
	client := openClient(t, "entrepo_session")
	ctx := context.Background()

	// Without WithNewID: ent's schema DefaultFunc generates the uuid, and the
	// generic impl copies it back.
	repo := New[Session, string](client)
	s := &Session{Note: "hi"}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := uuid.Parse(s.ID); err != nil {
		t.Errorf("ID %q is not a uuid (DefaultFunc + copy-back): %v", s.ID, err)
	}

	// With WithNewID and a preset ID: the preset wins via SetID.
	repo2 := New[Session, string](client, repository.WithNewID(uuid.NewString))
	pre := &Session{ID: "preset", Note: "keep"}
	if err := repo2.Create(ctx, pre); err != nil {
		t.Fatalf("Create preset: %v", err)
	}
	if pre.ID != "preset" {
		t.Errorf("ID = %q, want preset kept", pre.ID)
	}
	got, err := repo2.Get(ctx, "preset")
	if err != nil {
		t.Fatalf("Get preset: %v", err)
	}
	if got.Note != "keep" {
		t.Errorf("Note = %q, want keep", got.Note)
	}
}

func TestErrorLabels(t *testing.T) {
	type Widget struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	client := openClient(t, "entrepo_errors")
	repo := New[Widget, uint](client)

	_, err := repo.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for missing row")
	}
	if !strings.HasPrefix(err.Error(), "get widget by id:") {
		t.Errorf("error = %q, want prefix 'get widget by id:'", err)
	}
	var nf *entfix.NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("expected wrapped *ent.NotFoundError, got %v", err)
	}
}

func TestUnknownFieldError(t *testing.T) {
	// Widget's schema has only "name"; the extra tagged field must surface
	// ent's own SetField validation error, clearly labeled.
	type Widget struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Bogus string `json:"bogus"`
	}
	client := openClient(t, "entrepo_unknown")
	repo := New[Widget, uint](client)

	err := repo.Create(context.Background(), &Widget{Name: "x", Bogus: "y"})
	if err == nil {
		t.Fatal("expected SetField error for unknown field")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error = %q, want mention of the unknown field 'bogus'", err)
	}
}

func TestConstructionPanics(t *testing.T) {
	type Missing struct {
		ID uint `json:"id"`
	}
	client := openClient(t, "entrepo_panics")

	assertPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}
	assertPanic("no sub-client", func() { New[Missing, uint](client) })
	assertPanic("nil client", func() {
		type Widget struct {
			ID uint `json:"id"`
		}
		New[Widget, uint](nil)
	})
}
