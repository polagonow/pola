# openapi

Generate an **OpenAPI 3** specification from pola's discovered API routes and
serve it (with SwaggerUI) — a machine-readable contract for external consumers
(mobile apps, third parties, client codegen), complementing the typed
`@pola/actions` bridge that only served pola's own frontend.

```go
specs, _ := routes.Discover(reg)
doc := openapi.Generate(specs, openapi.Info{Title: "Shop API", Version: "1.0.0"})
reg.AddMiddleware(openapi.Serve(doc)) // /openapi.json + SwaggerUI at /openapi
```

## What it derives

- **Paths** and **methods** from every discovered route (`/users/:id` →
  `/users/{id}`).
- **Path parameters** (including catch-all `:...rest` → `{rest}`).
- **Summary** and **tags** per route from its `Meta()` (the routes package's
  `Metaer` interface):

  ```go
  func (r *UserRoutes) Meta() map[string]any {
      return map[string]any{"summary": "Manage users", "tags": []string{"users"}}
  }
  ```

## Scope

It documents the **API surface** (endpoints + params), not request/response body
schemas — pola routes don't yet declare their DTO types to the router. As routes
gain schema metadata (e.g. the DTO types in `Meta`), `Generate` can enrich
operations with request/response schemas; the response-schema side is a pola
advantage over generators that can only infer from validation, because the DTOs
(`dto/`) describe responses exactly.

`Serve` short-circuits its own two paths and passes everything else through, so —
like `middleware/health` — it needs no pipeline wiring.
