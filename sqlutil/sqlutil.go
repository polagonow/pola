// Package sqlutil holds small, dependency-free SQL helpers shared across pola's
// repository adapters.
package sqlutil

import "strings"

// LikeEscape is the escape character to pair with EscapeLike, e.g.
// db.Where("name LIKE ? ESCAPE '\\'", "%"+sqlutil.EscapeLike(q)+"%").
const LikeEscape = '\\'

// EscapeLike escapes the LIKE wildcards % and _ (and the escape character
// itself) in s so a user-supplied string is matched literally inside a LIKE
// pattern. Without this, a user typing "100%" would match every row. Pair the
// resulting pattern with an `ESCAPE '\'` clause so the backslash is honored on
// every driver.
func EscapeLike(s string) string {
	if !strings.ContainsAny(s, `\%_`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteByte(byte(LikeEscape))
		}
		b.WriteRune(r)
	}
	return b.String()
}
