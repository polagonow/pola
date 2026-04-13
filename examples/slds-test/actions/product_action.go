package actions

import (
	"context"

	"slds-test/repositories"
	"slds-test/services"
)

// ProductAction exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → ProductAction.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Usage in React:
//   import { ProductAction } from "@pola/actions"
//   const items = await ProductAction.list(1, 25)
//
// Run "pola generate" to regenerate the bridge after editing this file.

// ProductAction is your action struct with service dependency.
type ProductAction struct {
	svc *services.ProductService
}

// NewProductAction creates a ProductAction with its service dependency.
func NewProductAction(svc *services.ProductService) *ProductAction {
	return &ProductAction{svc: svc}
}

// List returns a paginated list of Product entities. Becomes ProductAction.list(page, perPage) in JavaScript.
func (a *ProductAction) List(page, perPage int) (*repositories.ListResult[*repositories.Product], error) {
	return a.svc.List(context.TODO(), repositories.ListParams{Page: page, PerPage: perPage})
}

// Get returns a single Product by ID. Becomes ProductAction.Get(id) in JavaScript.
func (a *ProductAction) Get(id uint) (*repositories.Product, error) {
	return a.svc.Get(context.TODO(), id)
}
