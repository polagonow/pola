// Package patternmatch provides URL pattern compilation and matching shared by
// both the page router and the API route router.
package patternmatch

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// SegmentKind describes the specificity of a single route segment.
// Lower value = more specific (used for segment-by-segment sorting).
type SegmentKind int

const (
	SegStatic      SegmentKind = 0 // e.g. "posts"
	SegDynamic     SegmentKind = 1 // e.g. ":slug"
	SegCatchAll    SegmentKind = 2 // e.g. ":...path"
	SegOptCatchAll SegmentKind = 3 // e.g. ":...slug?"
)

// CompiledRoute wraps a pattern with a pre-compiled regex and metadata
// for fast matching and correct sorting.
type CompiledRoute struct {
	Pattern    string
	Re         *regexp.Regexp
	ParamNames []string      // ordered capture group names
	ParamKinds []SegmentKind // parallel to ParamNames (for catch-all detection)
	IsStatic   bool          // true when pattern has no dynamic segments
	Segments   []SegmentKind // per-segment kind (excludes leading empty)
}

// CompilePattern converts a route pattern into a CompiledRoute with a
// pre-built regex, parameter metadata, and segment kinds for sorting.
func CompilePattern(pattern string) *CompiledRoute {
	if pattern == "/" {
		return &CompiledRoute{
			Pattern:  pattern,
			Re:       regexp.MustCompile(`^/$`),
			IsStatic: true,
			Segments: []SegmentKind{SegStatic},
		}
	}

	segs := strings.Split(pattern, "/")
	var reStr strings.Builder
	reStr.WriteString("^")

	var (
		paramNames []string
		paramKinds []SegmentKind
		segments   []SegmentKind
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
				segments = append(segments, SegOptCatchAll)
				paramKinds = append(paramKinds, SegOptCatchAll)
			} else {
				reStr.WriteString(`/(.+)`)
				segments = append(segments, SegCatchAll)
				paramKinds = append(paramKinds, SegCatchAll)
			}
			paramNames = append(paramNames, name)
			break // catch-all consumes the rest
		}

		reStr.WriteString("/")
		if strings.HasPrefix(seg, ":") {
			isStatic = false
			reStr.WriteString(`([^/]+)`)
			paramNames = append(paramNames, seg[1:])
			paramKinds = append(paramKinds, SegDynamic)
			segments = append(segments, SegDynamic)
		} else {
			reStr.WriteString(regexp.QuoteMeta(seg))
			segments = append(segments, SegStatic)
		}
	}

	reStr.WriteString(`(?:/)?$`)

	return &CompiledRoute{
		Pattern:    pattern,
		Re:         regexp.MustCompile(reStr.String()),
		ParamNames: paramNames,
		ParamKinds: paramKinds,
		IsStatic:   isStatic,
		Segments:   segments,
	}
}

// Match runs the compiled regex against path and extracts parameters.
// Catch-all params are returned as []string; dynamic params as decoded strings.
func Match(cr *CompiledRoute, path string) (map[string]any, bool) {
	m := cr.Re.FindStringSubmatch(path)
	if m == nil {
		return nil, false
	}
	params := make(map[string]any, len(cr.ParamNames))
	for i, name := range cr.ParamNames {
		val := m[i+1]
		if val == "" {
			continue // optional catch-all with no match
		}
		decoded, err := url.PathUnescape(val)
		if err != nil {
			decoded = val
		}
		if cr.ParamKinds[i] >= SegCatchAll {
			params[name] = strings.Split(decoded, "/")
		} else {
			params[name] = decoded
		}
	}
	return params, true
}

// SortBySpecificity sorts compiled routes by segment-by-segment specificity.
// Static segments beat dynamic, which beat catch-all, which beat optional catch-all.
// At equal specificity, longer routes win, then alphabetical tiebreak.
func SortBySpecificity(routes []*CompiledRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i].Segments, routes[j].Segments
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

// NormalizePath strips a single trailing slash (unless root "/").
func NormalizePath(path string) string {
	if len(path) > 1 && path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}
	return path
}
