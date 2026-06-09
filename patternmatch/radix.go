package patternmatch

import (
	"net/url"
	"strings"
)

// RouteTable is a radix-tree-based routing table that matches URL paths
// against compiled route patterns in O(log n) time for typical route sets.
type RouteTable struct {
	root *radixNode
}

// radixNode is a single node in the radix tree.
//
// The tree is structured as a segment-level trie where each level represents
// one path segment (the part between slashes). At each segment level:
//   - Static children are stored in a map keyed by the literal segment text
//   - A wildcard child matches any single segment (dynamic parameter)
//   - Catch-all and optional catch-all consume all remaining segments
//
// Priority: static > dynamic > catch-all > optional catch-all.
type radixNode struct {
	// staticChildren maps literal segment text to child nodes.
	staticChildren map[string]*radixNode

	// wildChild is the child for a dynamic segment (:name).
	wildChild *radixNode
	// wildName is the parameter name for the wildcard.
	wildName string

	// catchAll is a catch-all route (:...name) attached at this node.
	catchAll *CompiledRoute
	// catchAllName is the parameter name for the catch-all.
	catchAllName string

	// optCatchAll is an optional catch-all route (:...name?) at this node.
	optCatchAll *CompiledRoute
	// optCatchAllName is the parameter name for the optional catch-all.
	optCatchAllName string

	// route is set when this node is the terminal for a route.
	route *CompiledRoute
}

// NewRouteTable builds a segment-level trie from compiled routes.
func NewRouteTable(routes []*CompiledRoute) *RouteTable {
	// Sort by specificity so that when we insert, the first route to claim
	// a slot wins (static > dynamic > catch-all > optional catch-all).
	sorted := make([]*CompiledRoute, len(routes))
	copy(sorted, routes)
	SortBySpecificity(sorted)

	rt := &RouteTable{root: &radixNode{}}
	for _, cr := range sorted {
		rt.insert(cr)
	}
	return rt
}

// segment represents one parsed piece of a route pattern.
type segment struct {
	kind  SegmentKind
	value string // literal text for static; param name for dynamic/catch-all
}

// parseSegments splits a pattern like "/posts/:slug/:...rest" into segments.
func parseSegments(pattern string) []segment {
	if pattern == "/" {
		return nil // root route — empty segment list
	}
	parts := strings.Split(pattern, "/")
	var segs []segment
	for _, p := range parts[1:] { // skip leading empty string from split
		if strings.HasPrefix(p, ":...") {
			optional := strings.HasSuffix(p, "?")
			name := p[4:]
			if optional {
				name = name[:len(name)-1]
				segs = append(segs, segment{kind: SegOptCatchAll, value: name})
			} else {
				segs = append(segs, segment{kind: SegCatchAll, value: name})
			}
			break // catch-all consumes the rest
		}
		if strings.HasPrefix(p, ":") {
			segs = append(segs, segment{kind: SegDynamic, value: p[1:]})
		} else {
			segs = append(segs, segment{kind: SegStatic, value: p})
		}
	}
	return segs
}

// insert adds a compiled route into the trie.
func (rt *RouteTable) insert(cr *CompiledRoute) {
	segs := parseSegments(cr.Pattern)
	n := rt.root

	for i, seg := range segs {
		switch seg.kind {
		case SegStatic:
			if n.staticChildren == nil {
				n.staticChildren = make(map[string]*radixNode)
			}
			child, ok := n.staticChildren[seg.value]
			if !ok {
				child = &radixNode{}
				n.staticChildren[seg.value] = child
			}
			n = child

		case SegDynamic:
			if n.wildChild == nil {
				n.wildChild = &radixNode{}
				n.wildName = seg.value
			}
			n = n.wildChild

		case SegCatchAll:
			// Catch-all is always the last segment.
			if n.catchAll == nil {
				n.catchAll = cr
				n.catchAllName = seg.value
			}
			return

		case SegOptCatchAll:
			// Optional catch-all is always the last segment.
			if n.optCatchAll == nil {
				n.optCatchAll = cr
				n.optCatchAllName = seg.value
			}
			return
		}

		// If this is the last segment (and it was static or dynamic), set route.
		if i == len(segs)-1 {
			if n.route == nil {
				n.route = cr
			}
		}
	}

	// If segs was empty (root "/"), set route on root.
	if len(segs) == 0 {
		if n.route == nil {
			n.route = cr
		}
	}
}

// Match finds the best matching route for the given URL path.
// It returns the matched route, extracted parameters, and whether a match was found.
// Static segments take priority over dynamic, which take priority over catch-all.
func (rt *RouteTable) Match(path string) (route *CompiledRoute, params map[string]any, ok bool) {
	path = NormalizePath(path)
	p := make(map[string]any)
	r, matched := rt.matchNode(rt.root, splitSegments(path), p)
	if matched {
		return r, p, true
	}
	return nil, nil, false
}

// splitSegments splits a normalized path into its constituent segments.
// "/" => [] (empty), "/posts" => ["posts"], "/posts/new" => ["posts", "new"].
func splitSegments(path string) []string {
	if path == "/" || path == "" {
		return nil
	}
	return strings.Split(path[1:], "/") // strip leading '/'
}

// matchNode recursively searches the trie for a match.
// Priority: static children > wildcard child > catch-all > optional catch-all.
func (rt *RouteTable) matchNode(n *radixNode, segments []string, params map[string]any) (*CompiledRoute, bool) {
	// Base case: no more segments to consume.
	if len(segments) == 0 {
		if n.route != nil {
			return n.route, true
		}
		// Check optional catch-all (matches zero segments).
		if n.optCatchAll != nil {
			return n.optCatchAll, true
		}
		return nil, false
	}

	seg := segments[0]
	rest := segments[1:]

	// 1. Try static child (highest priority).
	if n.staticChildren != nil {
		if child, ok := n.staticChildren[seg]; ok {
			if r, ok := rt.matchNode(child, rest, params); ok {
				return r, true
			}
		}
	}

	// 2. Try wildcard child (dynamic parameter).
	if n.wildChild != nil {
		decoded, err := url.PathUnescape(seg)
		if err != nil {
			decoded = seg
		}
		// Save and set param.
		oldVal, hadKey := params[n.wildName]
		params[n.wildName] = decoded

		if r, ok := rt.matchNode(n.wildChild, rest, params); ok {
			return r, true
		}

		// Backtrack.
		if hadKey {
			params[n.wildName] = oldVal
		} else {
			delete(params, n.wildName)
		}
	}

	// 3. Try catch-all (consumes all remaining segments, mandatory >= 1).
	if n.catchAll != nil {
		parts := make([]string, len(segments))
		for i, s := range segments {
			d, err := url.PathUnescape(s)
			if err != nil {
				d = s
			}
			parts[i] = d
		}
		params[n.catchAllName] = parts
		return n.catchAll, true
	}

	// 4. Try optional catch-all (consumes all remaining segments, >= 0).
	if n.optCatchAll != nil {
		parts := make([]string, len(segments))
		for i, s := range segments {
			d, err := url.PathUnescape(s)
			if err != nil {
				d = s
			}
			parts[i] = d
		}
		params[n.optCatchAllName] = parts
		return n.optCatchAll, true
	}

	return nil, false
}
