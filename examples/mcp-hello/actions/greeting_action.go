package actions

import (
	"context"

	"github.com/polagonow/pola/core"
	"mcp-hello/repositories"
	"mcp-hello/services"

	"github.com/polagonow/pola/repository"
)

// GreetingAction exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → GreetingAction.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Usage in React:
//   import { GreetingAction } from "@pola/actions"
//   const items = await GreetingAction.list(1, 25)
//
// Run "pola generate" to regenerate the bridge after editing this file.

// GreetingAction is your action struct with service dependency.
type GreetingAction struct {
	svc *services.GreetingService
}

// NewGreetingAction creates a GreetingAction, resolving its dependencies from
// the DI registry.
func NewGreetingAction(r *core.Registry) *GreetingAction {
	return &GreetingAction{svc: core.MustInvoke[*services.GreetingService](r)}
}

// List returns a paginated list of Greeting entities. Becomes GreetingAction.list(page, perPage) in JavaScript.
func (a *GreetingAction) List(page, perPage int) (*repository.ListResult[*repositories.Greeting], error) {
	return a.svc.List(context.TODO(), repository.ListParams{Page: page, PerPage: perPage})
}

// Get returns a single Greeting by ID. Becomes GreetingAction.Get(id) in JavaScript.
func (a *GreetingAction) Get(id uint) (*repositories.Greeting, error) {
	return a.svc.Get(context.TODO(), id)
}
