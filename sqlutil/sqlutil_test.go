package sqlutil

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"plain":     "plain",
		"100%":      `100\%`,
		"a_b":       `a\_b`,
		`c\d`:       `c\\d`,
		"%_\\":      `\%\_\\`,
		"":          "",
		"no wilds!": "no wilds!",
	}
	for in, want := range cases {
		if got := EscapeLike(in); got != want {
			t.Errorf("EscapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}
