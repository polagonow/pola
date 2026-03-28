package actions

// Post exposes functions and variables to JavaScript via the Pola bridge.
//
// How it works:
//   - Each exported method becomes a callable function in JS
//   - Method names are automatically camelCased: GetItems → Post.getItems()
//   - Parameters are positional and type-safe (string, int, bool, structs)
//   - Return must be (T, error) or just error
//   - The special Vars() method exposes read-only constants to JS
//
// Run "pola generate" to regenerate the bridge after editing this file.

// Post is your action struct. Add fields for dependencies (DB, services, etc.)
// that will be resolved via dependency injection.
type Post struct {
	// DB *sql.DB  // example: injected database connection
}

// Example: a simple item type returned by your methods.
// Struct fields map to TypeScript interfaces via json tags.
// type Item struct {
// 	ID    int    `json:"id"`
// 	Name  string `json:"name"`
// }

// GetAll returns all items. Becomes Post.getAll() in JavaScript.
//
// Usage in React:
//   import { Post } from "@pola/di"
//   const items = await Post.getAll()
func (a *Post) GetAll() ([]map[string]any, error) {
	// TODO: implement
	return nil, nil
}

// GetByID returns a single item by ID. Becomes Post.getByID(id) in JavaScript.
//
// Parameters are positional in JS: Post.getByID("123")
func (a *Post) GetByID(id string) (map[string]any, error) {
	// TODO: implement
	return nil, nil
}

// Vars exposes read-only values to JavaScript as properties on the action.
// Each key becomes accessible as Post.keyName in JS.
//
// Use for: config values, feature flags, app name, environment info, etc.
// These are injected once per VM init, not per-request.
//
// func (a *Post) Vars() map[string]any {
// 	return map[string]any{
// 		"appName": "My App",
// 	}
// }
