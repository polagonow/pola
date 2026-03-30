package core

import "context"

// AppBuilder constructs a Pola application from plugins and options.
// Use NewAppBuilder to create one, then chain .Use() and .Build().
type AppBuilder struct {
	cfg     Config
	plugins []Plugin
}

// NewAppBuilder creates a new AppBuilder with the given options.
func NewAppBuilder(opts ...Option) *AppBuilder {
	b := &AppBuilder{}
	for _, opt := range opts {
		opt(&b.cfg)
	}
	return b
}

// Use adds plugins to the builder. Returns the builder for chaining.
func (b *AppBuilder) Use(plugins ...Plugin) *AppBuilder {
	b.plugins = append(b.plugins, plugins...)
	return b
}

// Config returns the resolved configuration.
func (b *AppBuilder) Config() *Config { return &b.cfg }

// Plugins returns the registered plugins.
func (b *AppBuilder) Plugins() []Plugin { return b.plugins }

// BuildRegistry creates a Registry and registers all plugins.
// This is called by the internal pipeline — not by end users directly.
func (b *AppBuilder) BuildRegistry(ctx context.Context) (*Registry, error) {
	reg := NewRegistry()

	// Register all plugins.
	for _, p := range b.plugins {
		p.Register(reg)
	}

	// Run lifecycle: Start
	for _, p := range b.plugins {
		if s, ok := p.(Starter); ok {
			if err := s.Start(ctx); err != nil {
				return nil, err
			}
		}
	}

	return reg, nil
}

// NotifyReady runs the Ready lifecycle hook on all plugins.
func (b *AppBuilder) NotifyReady(ctx context.Context) error {
	for _, p := range b.plugins {
		if rd, ok := p.(Readier); ok {
			if err := rd.Ready(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// Shutdown runs the Shutdown lifecycle hook on all plugins in reverse order.
func (b *AppBuilder) Shutdown(ctx context.Context) error {
	for i := len(b.plugins) - 1; i >= 0; i-- {
		if s, ok := b.plugins[i].(Shutdowner); ok {
			if err := s.Shutdown(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
