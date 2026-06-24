package repositories

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
