package nextjs

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/polagonow/pola/core"
)

// ── compiled route infrastructure ────────────────────────────────────────────

// segmentKind describes the specificity of a single route segment.
// Lower value = more specific (used for segment-by-segment sorting).
type segmentKind int

const (
	segStatic      segmentKind = 0 // e.g. "posts"
	segDynamic     segmentKind = 1 // e.g. ":slug"
	segCatchAll    segmentKind = 2 // e.g. ":...path"
	segOptCatchAll segmentKind = 3 // e.g. ":...slug?"
)

// compiledRoute wraps a core.Route with a pre-compiled regex and metadata
// for fast matching and correct sorting.
type compiledRoute struct {
	core.Route
	re         *regexp.Regexp
	paramNames []string      // ordered capture group names
	paramKinds []segmentKind // parallel to paramNames (for catch-all detection)
	isStatic   bool          // true when pattern has no dynamic segments
	segments   []segmentKind // per-segment kind (excludes leading empty)
}

// compilePattern converts a route pattern into a compiledRoute with a
// pre-built regex, parameter metadata, and segment kinds for sorting.
func compilePattern(route core.Route) *compiledRoute {
	pattern := route.Pattern
	if pattern == "/" {
		return &compiledRoute{
			Route:    route,
			re:       regexp.MustCompile(`^/$`),
			isStatic: true,
			segments: []segmentKind{segStatic},
		}
	}

	segs := strings.Split(pattern, "/")
	var reStr strings.Builder
	reStr.WriteString("^")

	var (
		paramNames []string
		paramKinds []segmentKind
		segments   []segmentKind
		isStatic   = true
	)

	for _, seg := range segs[1:] { // skip leading empty from split
		if strings.HasPrefix(seg, ":...") {
			isStatic = false
			optional := strings.HasSuffix(seg, "?")
			name := seg[4:]
			if optional {
				name = name[:len(name)-1]
				reStr.WriteString(`(?:/(.+))?`)
				segments = append(segments, segOptCatchAll)
				paramKinds = append(paramKinds, segOptCatchAll)
			} else {
				reStr.WriteString(`/(.+)`)
				segments = append(segments, segCatchAll)
				paramKinds = append(paramKinds, segCatchAll)
			}
			paramNames = append(paramNames, name)
			break // catch-all consumes the rest
		}

		reStr.WriteString("/")
		if strings.HasPrefix(seg, ":") {
			isStatic = false
			reStr.WriteString(`([^/]+)`)
			paramNames = append(paramNames, seg[1:])
			paramKinds = append(paramKinds, segDynamic)
			segments = append(segments, segDynamic)
		} else {
			reStr.WriteString(regexp.QuoteMeta(seg))
			segments = append(segments, segStatic)
		}
	}

	reStr.WriteString(`(?:/)?$`)

	return &compiledRoute{
		Route:      route,
		re:         regexp.MustCompile(reStr.String()),
		paramNames: paramNames,
		paramKinds: paramKinds,
		isStatic:   isStatic,
		segments:   segments,
	}
}

// matchCompiled runs the compiled regex against path and extracts parameters.
// Catch-all params are returned as []string; dynamic params as decoded strings.
func matchCompiled(cr *compiledRoute, path string) (map[string]any, bool) {
	m := cr.re.FindStringSubmatch(path)
	if m == nil {
		return nil, false
	}
	params := make(map[string]any, len(cr.paramNames))
	for i, name := range cr.paramNames {
		val := m[i+1]
		if val == "" {
			continue // optional catch-all with no match
		}
		decoded, err := url.PathUnescape(val)
		if err != nil {
			decoded = val
		}
		if cr.paramKinds[i] >= segCatchAll {
			params[name] = strings.Split(decoded, "/")
		} else {
			params[name] = decoded
		}
	}
	return params, true
}

// sortCompiledRoutes sorts compiled routes by segment-by-segment specificity.
// Static segments beat dynamic, which beat catch-all, which beat optional catch-all.
// At equal specificity, longer routes win, then alphabetical tiebreak.
func sortCompiledRoutes(routes []*compiledRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i].segments, routes[j].segments
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		for k := 0; k < minLen; k++ {
			if a[k] != b[k] {
				return a[k] < b[k] // lower kind = more specific
			}
		}
		if len(a) != len(b) {
			return len(a) > len(b) // longer = more specific
		}
		return routes[i].Pattern < routes[j].Pattern // alphabetical tiebreak
	})
}

// normalizePath strips a single trailing slash (unless root "/").
func normalizePath(path string) string {
	if len(path) > 1 && path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}
	return path
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
