package di

import (
	"sync"

	"github.com/samber/do/v2"
)

var (
	mu     sync.Mutex
	staged []func(do.Injector)
)

// Stage queues a provider registration closure to be replayed when Build()
// creates the injector. Designed to be called from init() in plugin packages.
func Stage(fn func(do.Injector)) {
	mu.Lock()
	defer mu.Unlock()
	staged = append(staged, fn)
}
