package server

import (
	"sort"
	"sync"

	"gojsx/framework/contract"
)

// PriorityRouter implements framework.Router using specificity-scored pattern
// matching. More-specific patterns (more static segments) are tried first.
type PriorityRouter struct {
	routes   []contract.Route
	sortOnce sync.Once
}

// Register adds a route to the router.
func (r *PriorityRouter) Register(route contract.Route) {
	r.routes = append(r.routes, route)
}

// Match returns the first route whose pattern matches path, plus captured params.
// Routes are sorted by specificity on the first call.
func (r *PriorityRouter) Match(path string) (*contract.Route, map[string]any) {
	r.sortOnce.Do(func() {
		sort.SliceStable(r.routes, func(i, j int) bool {
			return routeScore(r.routes[i].Pattern) > routeScore(r.routes[j].Pattern)
		})
	})
	for i := range r.routes {
		if params, ok := MatchPattern(r.routes[i].Pattern, path); ok {
			return &r.routes[i], params
		}
	}
	return nil, nil
}
