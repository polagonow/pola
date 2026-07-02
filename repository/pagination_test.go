package repository

import "testing"

func TestListParams_Normalize(t *testing.T) {
	cases := []struct {
		name string
		in   ListParams
		want ListParams
	}{
		{"defaults", ListParams{}, ListParams{Page: 1, PerPage: DefaultPerPage}},
		{"zero page", ListParams{Page: 0, PerPage: 50}, ListParams{Page: 1, PerPage: 50}},
		{"negative perPage", ListParams{Page: 3, PerPage: -1}, ListParams{Page: 3, PerPage: DefaultPerPage}},
		{"keep set values", ListParams{Page: 5, PerPage: 10}, ListParams{Page: 5, PerPage: 10}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.in.Normalize()
			if got != c.want {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

func TestListParams_Offset(t *testing.T) {
	cases := []struct {
		in   ListParams
		want int
	}{
		{ListParams{Page: 1, PerPage: 25}, 0},
		{ListParams{Page: 2, PerPage: 25}, 25},
		{ListParams{Page: 5, PerPage: 10}, 40},
	}
	for _, c := range cases {
		if got := c.in.Offset(); got != c.want {
			t.Fatalf("offset for %+v: got %d want %d", c.in, got, c.want)
		}
	}
}

func TestNewListResult(t *testing.T) {
	cases := []struct {
		name      string
		items     []int
		total     int
		params    ListParams
		wantPages int
	}{
		{"empty", nil, 0, ListParams{Page: 1, PerPage: 10}, 0},
		{"exact division", []int{1, 2}, 20, ListParams{Page: 1, PerPage: 10}, 2},
		{"remainder rounds up", []int{1, 2, 3}, 21, ListParams{Page: 3, PerPage: 10}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := NewListResult(c.items, c.total, c.params)
			if r.TotalPages != c.wantPages {
				t.Errorf("TotalPages = %d, want %d", r.TotalPages, c.wantPages)
			}
			if r.Total != c.total || r.Page != c.params.Page || r.PerPage != c.params.PerPage {
				t.Errorf("metadata = %+v, want total=%d page=%d perPage=%d", r, c.total, c.params.Page, c.params.PerPage)
			}
			if len(r.Items) != len(c.items) {
				t.Errorf("len(Items) = %d, want %d", len(r.Items), len(c.items))
			}
		})
	}
}

func TestEntityNameOf(t *testing.T) {
	type SampleEntity struct{}
	if got := EntityNameOf[SampleEntity](); got != "sample_entity" {
		t.Errorf("EntityNameOf[SampleEntity] = %q, want sample_entity", got)
	}
	type User struct{}
	if got := EntityNameOf[*User](); got != "user" {
		t.Errorf("EntityNameOf[*User] = %q, want user", got)
	}
}

func TestEnsureID(t *testing.T) {
	type E struct {
		ID   string
		Name string
	}

	e := &E{}
	EnsureID(e, func() string { return "generated" })
	if e.ID != "generated" {
		t.Errorf("ID = %q, want generated", e.ID)
	}

	pre := &E{ID: "keep"}
	EnsureID(pre, func() string { return "generated" })
	if pre.ID != "keep" {
		t.Errorf("ID = %q, want pre-set value kept", pre.ID)
	}

	EnsureID[E, string](nil, func() string { return "x" }) // no panic
	EnsureID[E, string](e, nil)                            // no panic
}

func TestMustIDFieldIndex_Panics(t *testing.T) {
	type NoID struct{ Name string }
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for entity without ID field")
		}
	}()
	MustIDFieldIndex[NoID, uint]()
}
