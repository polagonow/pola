package core

import "context"

// Plugin is a self-contained extension that provides services to the framework.
// Plugins register their services via the Registry during app construction.
type Plugin interface {
	Name() string
	Register(r *Registry)
}

// PluginFunc adapts a plain function to the Plugin interface.
type PluginFunc struct {
	PluginName string
	Fn         func(*Registry)
}

// Name returns the plugin name.
func (p PluginFunc) Name() string { return p.PluginName }

// Register calls the wrapped function.
func (p PluginFunc) Register(r *Registry) { p.Fn(r) }

// Starter is an optional interface for plugins that need to run setup logic
// after all services are registered but before the app is built.
type Starter interface {
	Start(ctx context.Context) error
}

// Readier is an optional interface for plugins that need to run logic
// after the app is fully wired and ready to serve.
type Readier interface {
	Ready(ctx context.Context) error
}

// Shutdowner is an optional interface for plugins that need cleanup on shutdown.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}
