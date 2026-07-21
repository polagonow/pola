package dto

import (
	"encoding/json"
	"testing"
)

func TestOptionalJSONUnmarshalPresence(t *testing.T) {
	type patch struct {
		Name  Optional[string]  `json:"name,omitzero"`
		Price Optional[float64] `json:"price,omitzero"`
	}

	var p patch
	if err := json.Unmarshal([]byte(`{"name":"widget"}`), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !p.Name.IsPresent() || p.Name.Val != "widget" {
		t.Errorf("Name = %+v, want present 'widget'", p.Name)
	}
	if p.Price.IsPresent() {
		t.Errorf("Price should be absent, got %+v", p.Price)
	}

	// An explicit null counts as present, decoding to the zero value.
	var p2 patch
	if err := json.Unmarshal([]byte(`{"price":null}`), &p2); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !p2.Price.IsPresent() || p2.Price.Val != 0 {
		t.Errorf("Price = %+v, want present 0", p2.Price)
	}

	// An explicit zero is present and distinct from absence.
	var p3 patch
	if err := json.Unmarshal([]byte(`{"price":0}`), &p3); err != nil {
		t.Fatalf("unmarshal zero: %v", err)
	}
	if !p3.Price.IsPresent() {
		t.Errorf("explicit 0 should be present")
	}
}

func TestOptionalJSONMarshalOmitzero(t *testing.T) {
	type patch struct {
		Name  Optional[string]  `json:"name,omitzero"`
		Price Optional[float64] `json:"price,omitzero"`
	}

	p := patch{Name: NewOptional("widget")}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(b), `{"name":"widget"}`; got != want {
		t.Errorf("marshal = %s, want %s (absent price omitted)", got, want)
	}

	empty, err := json.Marshal(patch{})
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if got, want := string(empty), `{}`; got != want {
		t.Errorf("marshal empty = %s, want %s", got, want)
	}
}

func TestOptionalHelpers(t *testing.T) {
	var o Optional[int]
	if o.IsPresent() || !o.IsZero() {
		t.Errorf("zero Optional should be absent")
	}
	if got := o.Default(42); got != 42 {
		t.Errorf("Default = %d, want 42", got)
	}

	o.Set(7)
	if v, ok := o.Get(); !ok || v != 7 {
		t.Errorf("Get = (%d,%v), want (7,true)", v, ok)
	}
	if got := o.Default(42); got != 7 {
		t.Errorf("Default after Set = %d, want 7", got)
	}

	o.Unset()
	if o.IsPresent() || o.Val != 0 {
		t.Errorf("Unset should clear value and presence, got %+v", o)
	}
}

func TestOptionalSQLValueScan(t *testing.T) {
	// Absent → SQL NULL.
	absent := Optional[int64]{}
	if v, err := absent.Value(); err != nil || v != nil {
		t.Errorf("absent.Value() = (%v,%v), want (nil,nil)", v, err)
	}

	// Present int converts through the standard driver conversion to int64.
	present := NewOptional(5)
	v, err := present.Value()
	if err != nil {
		t.Fatalf("present.Value(): %v", err)
	}
	if v != int64(5) {
		t.Errorf("present.Value() = %v (%T), want int64(5)", v, v)
	}

	// Scan a value → present; scan NULL → absent.
	var scanned Optional[int64]
	if err := scanned.Scan(int64(9)); err != nil {
		t.Fatalf("Scan(9): %v", err)
	}
	if !scanned.IsPresent() || scanned.Val != 9 {
		t.Errorf("after Scan(9) = %+v, want present 9", scanned)
	}
	if err := scanned.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if scanned.IsPresent() {
		t.Errorf("after Scan(nil) should be absent, got %+v", scanned)
	}
}

func TestConvertStripsUndeclaredFields(t *testing.T) {
	type userModel struct {
		ID       int64
		Name     string
		Password string
	}
	type userDTO struct {
		ID   int64  `json:"ID"`
		Name string `json:"Name"`
	}

	out, err := Convert[userDTO](userModel{ID: 1, Name: "ada", Password: "secret"})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if out.ID != 1 || out.Name != "ada" {
		t.Errorf("Convert = %+v, want {1 ada}", out)
	}
	// Password has no home on the DTO, so it is dropped — the round-trip is the
	// filter.
}

func TestMustConvertPanicsOnBadTarget(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("MustConvert should panic converting an object into an int")
		}
	}()
	_ = MustConvert[int](map[string]string{"a": "b"})
}

func TestCopyOptionalPartialUpdate(t *testing.T) {
	type Model struct {
		ID    int64
		Name  string
		Price int
		Tag   string
	}
	type UpdateDTO struct {
		Name  Optional[string]
		Price Optional[int64] // convertible to the model's int
		Tag   Optional[string]
	}

	model := &Model{ID: 1, Name: "old", Price: 10, Tag: "keep"}
	upd := UpdateDTO{}
	upd.Name.Set("new") // present
	upd.Tag.Set("")     // present, explicit zero
	// Price left absent.

	Copy(model, upd)

	if model.ID != 1 {
		t.Errorf("ID clobbered: %d", model.ID)
	}
	if model.Name != "new" {
		t.Errorf("Name = %q, want 'new' (present)", model.Name)
	}
	if model.Price != 10 {
		t.Errorf("Price = %d, want 10 (absent field untouched)", model.Price)
	}
	if model.Tag != "" {
		t.Errorf("Tag = %q, want '' (explicit zero written)", model.Tag)
	}
}

func TestCopyPlainSkipsZero(t *testing.T) {
	type Model struct {
		Name  string
		Price int
	}
	type CreateDTO struct {
		Name  string
		Price int
	}

	model := &Model{Name: "default", Price: 99}
	Copy(model, CreateDTO{Name: "given", Price: 0})

	if model.Name != "given" {
		t.Errorf("Name = %q, want 'given' (non-zero copied)", model.Name)
	}
	if model.Price != 99 {
		t.Errorf("Price = %d, want 99 (zero plain field skipped)", model.Price)
	}
}

func TestCopyTypeConversion(t *testing.T) {
	type Model struct{ Count int }
	type DTO struct{ Count Optional[int64] }

	model := &Model{}
	dto := DTO{}
	dto.Count.Set(20)
	Copy(model, dto)

	if model.Count != 20 {
		t.Errorf("Count = %d, want 20 (int64→int converted)", model.Count)
	}
}
