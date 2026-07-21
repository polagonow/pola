package repository

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseListQuery(t *testing.T) {
	v := url.Values{}
	v.Set("page", "2")
	v.Set("per_page", "20")
	v["filter"] = []string{
		"name||$cont||jack",
		"age||$between||18,30",
		"role||$in||admin,staff",
		"deleted_at||$isnull",
		"bad||$bogus||1", // unknown operator → dropped
		"||$eq||1",       // empty field → dropped
		"lonely",         // no operator → dropped
	}
	v["sort"] = []string{"created_at,desc", "name"}
	v.Set("fields", "id, name ,email")

	p := ParseListQuery(v)

	if p.Page != 2 || p.PerPage != 20 {
		t.Errorf("page/per_page = %d/%d, want 2/20", p.Page, p.PerPage)
	}
	if len(p.Filters) != 4 {
		t.Fatalf("filters = %d (%+v), want 4 valid", len(p.Filters), p.Filters)
	}
	if got := p.Filters[1]; got.Field != "age" || got.Operator != OpBetween || len(got.Args) != 2 {
		t.Errorf("between filter = %+v", got)
	}
	if got := p.Filters[2]; got.Operator != OpIn || len(got.Args) != 2 || got.Args[0] != "admin" {
		t.Errorf("in filter = %+v", got)
	}
	if got := p.Filters[3]; got.Operator != OpIsNull || len(got.Args) != 0 {
		t.Errorf("isnull filter = %+v", got)
	}
	if len(p.Sorts) != 2 || !p.Sorts[0].Desc || p.Sorts[1].Desc {
		t.Errorf("sorts = %+v", p.Sorts)
	}
	if !reflect.DeepEqual(p.Fields, []string{"id", "name", "email"}) {
		t.Errorf("fields = %v, want trimmed [id name email]", p.Fields)
	}
}

func TestParseListQueryBetweenRequiresTwoArgs(t *testing.T) {
	v := url.Values{"filter": []string{"age||$between||18"}}
	if p := ParseListQuery(v); len(p.Filters) != 0 {
		t.Errorf("between with one arg should be dropped, got %+v", p.Filters)
	}
}

func TestCoerce(t *testing.T) {
	if got := Coerce(reflect.TypeOf(0), "42"); got != int64(42) {
		t.Errorf("int coerce = %v (%T), want int64(42)", got, got)
	}
	if got := Coerce(reflect.TypeOf(0.0), "1.5"); got != 1.5 {
		t.Errorf("float coerce = %v, want 1.5", got)
	}
	if got := Coerce(reflect.TypeOf(true), "true"); got != true {
		t.Errorf("bool coerce = %v, want true", got)
	}
	if got := Coerce(reflect.TypeOf(""), "hello"); got != "hello" {
		t.Errorf("string coerce = %v, want hello", got)
	}
	// Non-numeric string for a numeric column falls back to the raw string
	// (the driver/DB then rejects or coerces it) rather than panicking.
	if got := Coerce(reflect.TypeOf(0), "notanumber"); got != "notanumber" {
		t.Errorf("bad int coerce = %v, want raw string fallback", got)
	}
}
