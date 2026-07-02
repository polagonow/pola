// Package pattern translates Pola's URL pattern syntax (":id" for a named param,
// ":...rest" for a catch-all) into the native syntax of each web framework.
//
// RESTful API routes derived by the routes package only ever use named params
// (e.g. "/users/:id/posts/:post_id"); catch-all support is best-effort for
// custom Path() overrides.
package pattern

import "strings"

// ToGin converts a Pola pattern to gin syntax: ":id" stays, ":...rest" → "*rest".
func ToGin(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if rest, ok := catchAll(s); ok {
			segs[i] = "*" + rest
		}
	}
	return strings.Join(segs, "/")
}

// ToEcho converts a Pola pattern to echo syntax: ":id" stays, ":...rest" → "*"
// (echo exposes the catch-all value under the param name "*").
func ToEcho(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if _, ok := catchAll(s); ok {
			segs[i] = "*"
		}
	}
	return strings.Join(segs, "/")
}

// ToChi converts a Pola pattern to chi syntax: ":id" → "{id}", ":...rest" → "*".
func ToChi(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if _, ok := catchAll(s); ok {
			segs[i] = "*"
		} else if strings.HasPrefix(s, ":") {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

// catchAll reports whether seg is a Pola catch-all (":...name") and returns the
// trailing name.
func catchAll(seg string) (string, bool) {
	if strings.HasPrefix(seg, ":...") {
		return seg[4:], true
	}
	return "", false
}
