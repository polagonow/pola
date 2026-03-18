package nextjs

import (
	"sort"
	"strings"

	"github.com/polagonow/pola/core"
)

// MatchPattern matches a URL path against a route pattern.
// Supports static segments, dynamic segments (":name"), catch-all (":...name"),
// and optional catch-all (":...name?").
func MatchPattern(pattern, path string) (map[string]any, bool) {
	pp := splitPath(pattern)
	rp := splitPath(path)
	params := map[string]any{}
	for i, seg := range pp {
		if strings.HasPrefix(seg, ":...") {
			optional := strings.HasSuffix(seg, "?")
			name := seg[4:]
			if optional {
				name = name[:len(name)-1]
			}
			remaining := rp[i:]
			if len(remaining) == 0 || (len(remaining) == 1 && remaining[0] == "") {
				if !optional {
					return nil, false
				}
				return params, true
			}
			params[name] = remaining
			return params, true
		}
		if i >= len(rp) {
			return nil, false
		}
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = rp[i]
		} else if seg != rp[i] {
			return nil, false
		}
	}
	if len(pp) != len(rp) {
		return nil, false
	}
	return params, true
}

func splitPath(p string) []string { return strings.Split(p, "/") }

// routeScore returns a priority score for a route pattern.
// Higher score = matched first. Static segments add score; catch-all
// segments subtract score; optional catch-all segments subtract more.
func routeScore(pattern string) int {
	score := 0
	for _, seg := range splitPath(pattern) {
		switch {
		case strings.HasPrefix(seg, ":...") && strings.HasSuffix(seg, "?"):
			score -= 20
		case strings.HasPrefix(seg, ":..."):
			score -= 10
		case strings.HasPrefix(seg, ":"):
			// neutral — dynamic segment does not change score
		default:
			score++
		}
	}
	return score
}

// sortRoutes sorts routes in-place: static routes first, then dynamic,
// then catch-all, then optional catch-all (highest score wins).
func sortRoutes(routes []core.Route) {
	sort.SliceStable(routes, func(i, j int) bool {
		return routeScore(routes[i].Pattern) > routeScore(routes[j].Pattern)
	})
}
