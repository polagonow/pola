package actions

import (
	"context"

	"blog-e2e-react/repositories"
	"blog-e2e-react/services"
)

// ArticleAction exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → ArticleAction.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Usage in React:
//   import { ArticleAction } from "@pola/actions"
//   const items = await ArticleAction.list(1, 25)
//
// Run "pola generate" to regenerate the bridge after editing this file.

// ArticleAction is your action struct with service dependency.
type ArticleAction struct {
	svc *services.ArticleService
}

// NewArticleAction creates a ArticleAction with its service dependency.
func NewArticleAction(svc *services.ArticleService) *ArticleAction {
	return &ArticleAction{svc: svc}
}

// List returns a paginated list of Article entities. Becomes ArticleAction.list(page, perPage) in JavaScript.
func (a *ArticleAction) List(page, perPage int) (*repositories.ListResult[*repositories.Article], error) {
	return a.svc.List(context.TODO(), repositories.ListParams{Page: page, PerPage: perPage})
}

// Get returns a single Article by ID. Becomes ArticleAction.Get(id) in JavaScript.
func (a *ArticleAction) Get(id uint) (*repositories.Article, error) {
	return a.svc.Get(context.TODO(), id)
}
