package actions

import (
	"context"

	"github.com/polagonow/pola/core"
	"blog-e2e-react/repositories"
	"blog-e2e-react/services"

	"github.com/polagonow/pola/repository"
)

// SampleEntityAction exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → SampleEntityAction.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Usage in React:
//   import { SampleEntityAction } from "@pola/actions"
//   const items = await SampleEntityAction.list(1, 25)
//
// Run "pola generate" to regenerate the bridge after editing this file.

// SampleEntityAction is your action struct with service dependency.
type SampleEntityAction struct {
	svc *services.SampleEntityService
}

// NewSampleEntityAction creates a SampleEntityAction, resolving its
// dependencies from the DI registry.
func NewSampleEntityAction(r *core.Registry) *SampleEntityAction {
	return &SampleEntityAction{svc: core.MustInvoke[*services.SampleEntityService](r)}
}

// List returns a paginated list of SampleEntity entities. Becomes SampleEntityAction.list(page, perPage) in JavaScript.
func (a *SampleEntityAction) List(page, perPage int) (*repository.ListResult[*repositories.SampleEntity], error) {
	return a.svc.List(context.TODO(), repository.ListParams{Page: page, PerPage: perPage})
}

// Get returns a single SampleEntity by ID. Becomes SampleEntityAction.Get(id) in JavaScript.
func (a *SampleEntityAction) Get(id uint) (*repositories.SampleEntity, error) {
	return a.svc.Get(context.TODO(), id)
}
