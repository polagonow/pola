package di

import samberdo "github.com/samber/do/v2"

// Build creates the root injector, registers framework-internal singletons,
// then replays all providers that plugins staged during init().
// No noop defaults are registered — if a required service is missing,
// the pipeline will error with a clear message telling the user what to import.
func Build() samberdo.Injector {
	mu.Lock()
	fns := make([]func(samberdo.Injector), len(staged))
	copy(fns, staged)
	mu.Unlock()

	i := samberdo.New()

	// Framework-internal singletons — always needed, plugins don't replace these.
	samberdo.Provide(i, func(_ samberdo.Injector) (*MiddlewareCollector, error) {
		return &MiddlewareCollector{}, nil
	})
	samberdo.Provide(i, func(_ samberdo.Injector) (*InjectorCollector, error) {
		return &InjectorCollector{}, nil
	})
	samberdo.Provide(i, func(_ samberdo.Injector) (*EventBus, error) {
		return NewEventBus(), nil
	})

	// Replay plugin registrations.
	for _, fn := range fns {
		fn(i)
	}

	return i
}
