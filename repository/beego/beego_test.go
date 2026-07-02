package beego

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/polagonow/pola/repository"
)

// Test entities carry the same orm tags the repository generator emits, so
// these tests exercise the exact shape of generated code. Each test that
// needs row-count isolation gets its own entity/table (beego's model cache
// and the sqlite database are shared across the package's tests).

type BWidget struct {
	ID   uint   `orm:"column(id);auto;pk"`
	Name string `orm:"column(name)"`
}

func (BWidget) TableName() string { return "b_widgets" }

type BSession struct {
	ID   string `orm:"column(id);pk;size(36)"`
	Note string `orm:"column(note)"`
}

func (BSession) TableName() string { return "b_sessions" }

type BListItem struct {
	ID   uint   `orm:"column(id);auto;pk"`
	Name string `orm:"column(name)"`
}

func (BListItem) TableName() string { return "b_list_items" }

// BManual is pre-registered directly with beego in TestMain to prove
// ensureRegistered tolerates models the app already registered.
type BManual struct {
	ID uint `orm:"column(id);auto;pk"`
}

func (BManual) TableName() string { return "b_manuals" }

func TestMain(m *testing.M) {
	if err := orm.RegisterDataBase("default", "sqlite3", "file:beegorepo?mode=memory&cache=shared"); err != nil {
		panic(err)
	}
	// The app pre-registers this one (migration-main style).
	orm.RegisterModel(new(BManual))
	// The rest go through the package's own registration path.
	ensureRegistered(new(BWidget))
	ensureRegistered(new(BSession))
	ensureRegistered(new(BListItem))
	if err := orm.RunSyncdb("default", false, false); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestBeegoCRUD(t *testing.T) {
	repo := New[BWidget, uint](orm.NewOrm())
	ctx := context.Background()

	w := &BWidget{Name: "alpha"}
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

func TestBeegoStringIDWithNewID(t *testing.T) {
	repo := New[BSession, string](orm.NewOrm(), repository.WithNewID(uuid.NewString))
	ctx := context.Background()

	s := &BSession{Note: "hi"}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := uuid.Parse(s.ID); err != nil {
		t.Errorf("ID %q is not a uuid: %v", s.ID, err)
	}

	got, err := repo.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get by uuid: %v", err)
	}
	if got.Note != "hi" {
		t.Errorf("Note = %q, want hi", got.Note)
	}
}

func TestBeegoListPagination(t *testing.T) {
	repo := New[BListItem, uint](orm.NewOrm())
	ctx := context.Background()

	for _, n := range []string{"a", "b", "c"} {
		if err := repo.Create(ctx, &BListItem{Name: n}); err != nil {
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
}

func TestBeegoErrorLabel(t *testing.T) {
	repo := New[BWidget, uint](orm.NewOrm())
	_, err := repo.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for missing row")
	}
	if !strings.HasPrefix(err.Error(), "get b_widget by id:") {
		t.Errorf("error = %q, want prefix 'get b_widget by id:'", err)
	}
}

func TestRegistrationIdempotence(t *testing.T) {
	// Constructing twice must not panic (sync.Map guard).
	_ = New[BWidget, uint](orm.NewOrm())
	_ = New[BWidget, uint](orm.NewOrm())

	// A model the app registered itself must not panic either (recover guard:
	// beego panics "repeat Register", ensureRegistered swallows exactly that).
	ensureRegistered(new(BManual))
	_ = New[BManual, uint](orm.NewOrm())
}
