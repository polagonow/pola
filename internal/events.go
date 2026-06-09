package internal

import "github.com/polagonow/pola/core"

// EventBundleRebuilt is published when the bundler watch delivers new output.
type EventBundleRebuilt struct {
	Output *core.BundleOutput
}

// EventRoutesChanged is published when a route re-scan detects changes.
type EventRoutesChanged struct {
	Routes []core.Route
}
