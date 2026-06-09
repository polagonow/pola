package nextjs

import (
	"sort"
	"strings"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/patternmatch"
)

// compiledRoute wraps a core.Route with the shared compiled pattern.
type compiledRoute struct {
	core.Route
	compiled *patternmatch.CompiledRoute
}

// compilePattern converts a core.Route into a compiledRoute.
func compilePattern(route core.Route) *compiledRoute {
	return &compiledRoute{
		Route:    route,
		compiled: patternmatch.CompilePattern(route.Pattern),
	}
}

// matchCompiled runs the compiled regex against path and extracts parameters.
func matchCompiled(cr *compiledRoute, path string) (map[string]any, bool) {
	return patternmatch.Match(cr.compiled, path)
}

// sortCompiledRoutes sorts compiled routes by segment-by-segment specificity.
func sortCompiledRoutes(routes []*compiledRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i].compiled.Segments, routes[j].compiled.Segments
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		for k := 0; k < minLen; k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		if len(a) != len(b) {
			return len(a) > len(b)
		}
		return routes[i].Pattern < routes[j].Pattern
	})
}

// normalizePath strips a single trailing slash (unless root "/").
func normalizePath(path string) string {
	return patternmatch.NormalizePath(path)
}

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
func routeScore(pattern string) int {
	score := 0
	for _, seg := range splitPath(pattern) {
		switch {
		case strings.HasPrefix(seg, ":...") && strings.HasSuffix(seg, "?"):
			score -= 20
		case strings.HasPrefix(seg, ":..."):
			score -= 10
		case strings.HasPrefix(seg, ":"):
			// neutral
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

// ── Radix tree route resolver ───────────────────────────────────────────────

// routeResolver wraps a patternmatch.RouteTable and maps CompiledRoute pointers
// back to the nextjs compiledRoute entries for efficient radix-tree lookups.
type routeResolver struct {
	table   *patternmatch.RouteTable
	byRoute map[*patternmatch.CompiledRoute]*compiledRoute
}

// buildRouteResolver constructs a routeResolver from a set of compiled routes.
// The returned resolver uses a radix tree for O(log n) path matching.
func buildRouteResolver(routes []*compiledRoute) *routeResolver {
	compiled := make([]*patternmatch.CompiledRoute, len(routes))
	lookup := make(map[*patternmatch.CompiledRoute]*compiledRoute, len(routes))
	for i, cr := range routes {
		compiled[i] = cr.compiled
		lookup[cr.compiled] = cr
	}
	return &routeResolver{
		table:   patternmatch.NewRouteTable(compiled),
		byRoute: lookup,
	}
}

// Match resolves a URL path to a compiledRoute and extracted parameters
// using the radix tree.
func (rr *routeResolver) Match(path string) (*compiledRoute, map[string]any, bool) {
	cr, params, ok := rr.table.Match(path)
	if !ok {
		return nil, nil, false
	}
	compiled, exists := rr.byRoute[cr]
	if !exists {
		return nil, nil, false
	}
	return compiled, params, true
}
