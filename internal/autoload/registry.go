package autoload

import "sort"

var registry = map[string]Autoload{}

// Register adds an autoload to the global registry.
func Register(a Autoload) {
	registry[a.Name()] = a
}

// All returns all registered autoloads sorted by priority.
func All() []Autoload {
	out := make([]Autoload, 0, len(registry))
	for _, a := range registry {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Priority() < out[j].Priority()
	})
	return out
}

// Get returns an autoload by name, or nil if not found.
func Get(name string) Autoload {
	return registry[name]
}
