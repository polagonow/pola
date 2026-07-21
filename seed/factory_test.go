package seed

import (
	"context"
	"errors"
	"testing"
)

type widget struct {
	ID   int
	Name string
	Tier string
}

func TestFactoryGenerate(t *testing.T) {
	n := 0
	f := NewFactory(func() *widget {
		n++
		return &widget{Name: "w", Tier: "free"}
	})

	got := f.Generate(3)
	if len(got) != 3 {
		t.Fatalf("Generate(3) len = %d, want 3", len(got))
	}
	if n != 3 {
		t.Errorf("generator called %d times, want 3", n)
	}
	if f.Generate(-1); len(f.Generate(-1)) != 0 {
		t.Errorf("Generate(-1) should yield an empty slice")
	}
}

func TestFactoryOverride(t *testing.T) {
	f := NewFactory(func() *widget { return &widget{Name: "w", Tier: "free"} }).
		Override(func(w *widget) { w.Tier = "pro" })

	for _, w := range f.Generate(2) {
		if w.Tier != "pro" {
			t.Errorf("override not applied: %+v", w)
		}
	}
}

func TestFactorySave(t *testing.T) {
	f := NewFactory(func() *widget { return &widget{Name: "w"} })

	var saved []*widget
	id := 0
	create := func(_ context.Context, w *widget) error {
		id++
		w.ID = id // emulate an ID assigned on insert
		saved = append(saved, w)
		return nil
	}

	got, err := f.Save(context.Background(), create, 3)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if len(saved) != 3 || len(got) != 3 {
		t.Fatalf("saved %d / returned %d, want 3/3", len(saved), len(got))
	}
	if got[0].ID == 0 {
		t.Error("Save should reflect IDs populated by create")
	}
}

func TestFactorySaveStopsOnError(t *testing.T) {
	f := NewFactory(func() *widget { return &widget{Name: "w"} })

	boom := errors.New("insert failed")
	calls := 0
	create := func(_ context.Context, _ *widget) error {
		calls++
		if calls == 2 {
			return boom
		}
		return nil
	}

	_, err := f.Save(context.Background(), create, 5)
	if !errors.Is(err, boom) {
		t.Fatalf("Save err = %v, want boom", err)
	}
	if calls != 2 {
		t.Errorf("create called %d times, want 2 (stopped at first error)", calls)
	}
}
