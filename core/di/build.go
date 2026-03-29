package di

import "github.com/samber/do/v2"

// Build creates the root injector, registers framework-internal singletons,
// then replays all providers that plugins staged during init().
// No noop defaults are registered — if a required service is missing,
// the pipeline will error with a clear message telling the user what to import.
func Build() do.Injector {
	mu.Lock()
	fns := make([]func(do.Injector), len(staged))
	copy(fns, staged)
	mu.Unlock()

	i := do.New()

	// Framework-internal singletons — always needed, plugins don't replace these.
	do.Provide(i, func(_ do.Injector) (*MiddlewareCollector, error) {
		return &MiddlewareCollector{}, nil
	})
	do.Provide(i, func(_ do.Injector) (*InjectorCollector, error) {
		return &InjectorCollector{}, nil
	})
	do.Provide(i, func(_ do.Injector) (*EventBus, error) {
		return NewEventBus(), nil
	})

	// Replay plugin registrations.
	for _, fn := range fns {
		fn(i)
	}

	return i
}
