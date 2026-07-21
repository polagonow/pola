# DTO

Framework-owned data-transfer types and mapping helpers for the presentation
layer. Keep database **models** in the data layer, expose **DTOs** to clients,
and move data across the boundary with three small, explicit tools. The design
borrows the shape proven by Goyave's `typeutil`, implemented dependency-free
(reflection + `encoding/json`) to fit pola's Go-native philosophy.

```go
type Product struct {           // response DTO — what clients see
    ID    int64   `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

type UpdateProduct struct {     // PATCH DTO — only supplied fields change
    Name  dto.Optional[string]  `json:"name,omitzero"`
    Price dto.Optional[float64] `json:"price,omitzero"`
}
```

## Why

pola binds request bodies straight into models, which loses two things:

1. **No PATCH semantics.** A plain `Price float64` can't tell "client sent
   `price: 0`" from "client omitted price" — both arrive as the zero value, so a
   naive update clobbers the column. `Optional[T]` records *presence* without
   pointers, so [`Copy`](./dto.go) touches only supplied fields.
2. **No layer separation.** Models leak persistence details (and sensitive
   columns) to the wire. [`Convert`](./dto.go) round-trips through JSON, keeping
   only the fields the DTO declares — a response DTO drops columns it omits, and
   unexpected client input is dropped rather than bound.

## The three tools

- **`Convert[T](src) (T, error)` / `MustConvert[T](src) T`** — turn any value (a
  model, a map, another DTO) into a typed DTO, retaining only declared fields.
- **`Copy(dst *M, src D) *M`** — write a DTO's fields onto a model for
  persistence: `Optional` fields only when present, plain fields only when
  non-zero, incompatible/unmatched fields skipped.
- **`Optional[T]`** — distinguishes absent from zero. Implements
  `json.Marshaler`/`Unmarshaler` (pair with `,omitzero`), `driver.Valuer` /
  `sql.Scanner` (via stdlib `sql.Null[T]`), and the presence protocol `Copy`
  reads. Helpers: `IsPresent`, `Get`, `Default`, `Set`, `Unset`.

## Recommended flow

```go
// Request → validated DTO → model → save → response DTO.
func (s *ProductService) Update(ctx context.Context, id int64, in dto.UpdateProduct) (dto.Product, error) {
    model, err := s.repo.Get(ctx, id) // fetch first (read-modify-write)
    if err != nil {
        return dto.Product{}, err
    }
    dto.Copy(model, in)               // apply only present fields
    if err := s.repo.Update(ctx, model); err != nil {
        return dto.Product{}, err
    }
    return dto.MustConvert[dto.Product](model), nil // strip non-DTO columns
}
```

Fetching the model before `Copy` (rather than building a fresh model from a
partial DTO) avoids temporal inconsistency and preserves untouched columns.
