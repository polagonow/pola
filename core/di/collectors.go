package di

import "github.com/polagonow/pola/core"

// MiddlewareCollector accumulates middleware registrations from multiple plugins.
type MiddlewareCollector struct {
	entries []core.Middleware
}

// Add appends a middleware to the collector.
func (c *MiddlewareCollector) Add(m core.Middleware) { c.entries = append(c.entries, m) }

// All returns all registered middleware in registration order.
func (c *MiddlewareCollector) All() []core.Middleware { return c.entries }

// InjectorCollector accumulates RuntimeInjector registrations from multiple plugins.
type InjectorCollector struct {
	entries []core.RuntimeInjector
}

// Add appends a RuntimeInjector to the collector.
func (c *InjectorCollector) Add(inj core.RuntimeInjector) { c.entries = append(c.entries, inj) }

// All returns all registered RuntimeInjectors in registration order.
func (c *InjectorCollector) All() []core.RuntimeInjector { return c.entries }
