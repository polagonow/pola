# Data layer: field grammar + the model → action chain

## Field specification grammar

Shared by `model`, `repository`, `service`, `scaffold`, `zod`, and `page`:

```
field:type{options}:modifier
```

**Types** (the complete set): `string`, `int`, `int64`, `float`, `bool`, `time`, `uuid`, `text`,
`bytes`, `json`, `references`.

**Trailing `?`** on the type makes the field optional/nullable: `bio:text?`, `age:int?`.

**`{…}` options:**
- `{ModelName}` — on `references`, points to a specific target model:
  `avatar:references{StorageBlob}`.
- `{polymorphic}` — on `references`, a polymorphic association (adds a type + id pair):
  `commentable:references{polymorphic}`.
- `{N}` — a size limit on sized types: `name:string{255}` → `varchar(255)`.

**Modifiers** (chain after `:`): `index`, `uniq`.

**Primary key:** by default the model gets an auto-increment integer PK (`uint`, surfaced as gorm's
embedded `gorm.Model`). A UUID PK is possible (string id) when the generator is told to use one.

**Examples:**
```bash
pola generate model User name:string email:string:uniq age:int?
pola generate model Article title:string:index body:text author:references
pola generate model Comment body:text commentable:references{polymorphic}
pola generate model Profile name:string{120} avatar:references{StorageBlob}
```

## The chain

`db/models` → `repositories/` (interface + ORM impl, paginated) → `services/` (business logic) →
`dto/` (request/response shapes) → `actions/` (React bridge) **or** `routes/` (HTTP).
`pola generate scaffold` produces all of it at once (model + repository + service + dto +
action + route + zod + pages); the individual generators produce one layer each. Below is the
verified shape of each layer (gorm ORM).

### 1. ORM model — `db/models/gorm/<name>.go`
```go
package gorm

import "gorm.io/gorm"

// SampleEntity represents the sample_entity database table.
type SampleEntity struct {
    gorm.Model                                       // ID, CreatedAt, UpdatedAt, DeletedAt
    Name string `gorm:"type:varchar(255)" json:"name"`
}
```
(With `database.orm = "ent"`, the model is an ent schema under `db/models/schema/<name>.go`
instead.)

### 2. Repository — interface + impl + pagination
Interface (`repositories/<name>_repository.go`) — note it carries a plain transport struct used
across the bridge:
```go
package repositories

import "context"

type SampleEntity struct {
    ID   uint   `json:"id"`
    Name string `json:"name" valid:"required"`
}

type SampleEntityRepository interface {
    Create(ctx context.Context, entity *SampleEntity) error
    Get(ctx context.Context, id uint) (*SampleEntity, error)
    List(ctx context.Context, params ListParams) (*ListResult[*SampleEntity], error)
    Update(ctx context.Context, entity *SampleEntity) error
    Delete(ctx context.Context, id uint) error
}
```
Implementation (`repositories/gorm/<name>_repository.go`):
```go
package gorm

import (
    "context"
    "fmt"

    "myapp/repositories"
    "gorm.io/gorm"
)

type sampleEntityRepository struct{ db *gorm.DB }

func NewSampleEntityRepository(db *gorm.DB) repositories.SampleEntityRepository {
    return &sampleEntityRepository{db: db}
}

func (r *sampleEntityRepository) Get(ctx context.Context, id uint) (*repositories.SampleEntity, error) {
    var entity repositories.SampleEntity
    if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
        return nil, fmt.Errorf("get sample_entity by id: %w", err)
    }
    return &entity, nil
}
// Create / List / Update / Delete follow the same pattern; List uses params.Normalize(),
// .Offset(params.Offset()), .Limit(params.PerPage) and returns a *ListResult[T].
```
Pagination/filtering types come from the framework package
`github.com/polagonow/pola/repository` (older scaffolds carried a local
`repositories/pagination.go` copy):
```go
type ListParams struct {
    Page    int      // clamped to >= 1 by Normalize()
    PerPage int      // DefaultPerPage = 25
    Filters []Filter // WHERE conditions
    Sorts   []Sort   // ORDER BY
    Fields  []string // SELECT column allowlist
}
func (p ListParams) Normalize() ListParams
func (p ListParams) Offset() int

type ListResult[T any] struct {
    Items      []T
    Total      int
    Page       int
    PerPage    int
    TotalPages int
}
func NewListResult[T any](items []T, total int, params ListParams) *ListResult[T]
```

### Query-driven filtering (`repository.ParseListQuery`)

`repository.ParseListQuery(r.URL.Query())` turns a Goyave-style query string into
`ListParams`; the generated ORM repositories apply it (with a per-entity column
allowlist, so unknown fields/operators are dropped rather than injected):

```
?page=2&per_page=20
&filter=name||$cont||jack          (repeatable; conditions are ANDed)
&filter=age||$between||18,30
&filter=role||$in||admin,staff
&sort=created_at,desc              (repeatable)
&fields=id,name,email
```

| Operator | Meaning | | Operator | Meaning |
|----------|---------|-|----------|---------|
| `$eq` / `$ne` | = / != | | `$cont` | `LIKE %x%` |
| `$gt` / `$gte` | > / >= | | `$starts` / `$ends` | prefix / suffix LIKE |
| `$lt` / `$lte` | < / <= | | `$in` / `$notin` | value list |
| `$between` | two args | | `$isnull` / `$notnull` | no args |

```go
// routes/products/route.go
params := repository.ParseListQuery(req.URL.Query())
result, err := r.svc.List(req.Context(), params)
```

`repository.ErrNotFound` / `repository.IsNotFound(err)` are the ORM-agnostic
not-found sentinels.

### 3. Service — `services/<name>_service.go`
```go
package services

import (
    "context"
    "myapp/repositories"
)

type SampleEntityService struct{ repo repositories.SampleEntityRepository }

func NewSampleEntityService(repo repositories.SampleEntityRepository) *SampleEntityService {
    return &SampleEntityService{repo: repo}
}

func (s *SampleEntityService) List(ctx context.Context, p repositories.ListParams) (*repositories.ListResult[*repositories.SampleEntity], error) {
    return s.repo.List(ctx, p)
}
// Add business logic (validation, authorization, side effects) here before delegating to repo.
```

### 4a. Action — `actions/<name>_action.go` (exposes the service to React)
```go
package actions

import (
    "context"
    "myapp/repositories"
    "myapp/services"
)

type ProductAction struct{ svc *services.ProductService }

func NewProductAction(svc *services.ProductService) *ProductAction { return &ProductAction{svc: svc} }

// Becomes ProductAction.list(page, perPage) in JS.
func (a *ProductAction) List(page, perPage int) (*repositories.ListResult[*repositories.Product], error) {
    return a.svc.List(context.TODO(), repositories.ListParams{Page: page, PerPage: perPage})
}

// Becomes ProductAction.get(id) in JS.
func (a *ProductAction) Get(id uint) (*repositories.Product, error) {
    return a.svc.Get(context.TODO(), id)
}
```
Consumed by a Server Component:
```tsx
import { ProductAction } from "@pola/actions";

export default async function ProductsPage({ searchParams }: { searchParams?: Record<string,string> }) {
  const page = parseInt(searchParams?.page || "1", 10);
  const result = await ProductAction.list(page, 25);
  return <ProductsListView items={result.items} />;
}
```

### 4b. Route — `routes/<path>/route.go` (HTTP/JSON endpoint, alternative to an action)
```go
package posts

import (
    "encoding/json"
    "net/http"
)

type Route struct{}

// GET /posts
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(/* … */)
}

// POST /posts
func (r *Route) POST(w http.ResponseWriter, req *http.Request) { /* … */ }
```
Route path derives from the directory: `routes/posts/route.go` → `/posts`,
`routes/posts/comments/route.go` → `/posts/comments`. Pass `--service` to wire a service in.

## DTOs (`github.com/polagonow/pola/dto`)

`pola generate dto <Name> [fields]` (also part of `scaffold`) writes `dto/<snake>.go`
with a response DTO plus `Create`/`Update` DTOs. The package API:

| Function | Purpose |
|----------|---------|
| `dto.Convert[T](src) (T, error)` / `MustConvert[T](src) T` | JSON round-trip conversion — response DTOs keep only declared fields (internal columns never leak) |
| `dto.Copy[M, D](dst *M, src D) *M` | DTO → model field mapping; skips fields tagged readonly and absent `Optional`s |
| `dto.Optional[T]` | Distinguishes *absent* from *zero value* — the Update DTO uses it so PATCH only touches fields the client actually sent (`Set/Unset/IsPresent/Get/Default`; marshals to/from JSON and sql driver values) |

```go
// PATCH handler: only present fields are copied onto the model
var in dto.UpdateProduct
json.NewDecoder(req.Body).Decode(&in)
model, _ := r.svc.Get(ctx, id)
dto.Copy(model, in)
r.svc.Update(ctx, model)
// response: dto.MustConvert[dto.Product](model)
```

## Transactions (`repository/uow`)

`uow.TxManager` is the ORM-neutral unit-of-work interface; `repository/gorm`
provides the implementation. Repositories auto-enlist in an ambient transaction
carried on the context, and nesting uses SAVEPOINTs:

```go
import (
    "github.com/polagonow/pola/repository/uow"
    repogorm "github.com/polagonow/pola/repository/gorm"
)

type BillingService struct{ tx uow.TxManager /* repogorm.NewTxManager(db) */ }

func (s *BillingService) Transfer(ctx context.Context, from, to uint, amount int64) error {
    return s.tx.Transaction(ctx, func(ctx context.Context) error {
        if err := s.accounts.Debit(ctx, from, amount); err != nil {
            return err // rolls back
        }
        return s.accounts.Credit(ctx, to, amount) // commits on nil
    })
}
```

## Seeding (`github.com/polagonow/pola/seed`)

`pola generate seed` scaffolds `db/seeds/seeds.go` with
`Seed(ctx context.Context, r *core.Registry) error`; `pola db seed` builds the app
with the seeder wired in and runs it (`POLA_SEED_ONLY=true` makes `pola.Ready()`
seed and exit). The seeder gets the full DI registry — resolve anything with
`core.MustInvoke`. Guard inserts so re-running doesn't duplicate data.

Batch data uses the factory:

```go
users := seed.NewFactory(func() *models.User {
    return &models.User{Name: gofakeit.Name(), Email: gofakeit.Email()}
})
users.Override(func(u *models.User) { u.TeamID = team.ID })
_, err := users.Save(ctx, repo.Create, 10) // Save(ctx, func(ctx, *T) error, n)
```

## Validation

Transport structs carry go-playground/validator tags; the framework exposes them via
`github.com/polagonow/pola/validation` — `Validate(v any) error` returns a
`ValidationErrors` list (field, tag, value, message), and
`ValidateField(field any, tag string)` checks a single value. Generated code uses
tags like `valid:"required"` on repository transport structs.

## ORM selection

`database.orm` in `Polafile.hcl` picks the templates: `gorm` (models under `db/models/gorm/`,
embeds `gorm.Model`) or `ent` (schemas under `db/models/schema/`). **When the Polafile
omits `orm`, the framework defaults to `ent`** — scaffolded apps write the choice
explicitly. The repository implementation package (`repositories/gorm/` vs
`repositories/ent/`) follows the same choice. Change ORM by editing the Polafile
before generating (this regenerates a different repository impl — a breaking change
for existing code).

## Migrations

- `pola generate scaffold`/`model` auto-emit a migration (unless `--skip-migration`).
- `pola generate migration <Name>` diffs the ORM models against `db/migrations/` using **Atlas** and
  writes a versioned file (`<timestamp>_<name>.sql` or `.hcl` per `migrations.format`) into
  `database.migrations.directory`.
- Each migration carries `-- atlas:down` directives; `pola db rollback` uses them.
- Apply with `pola db migrate`; inspect with `pola db status`; wipe-and-replay with `pola db reset`.
- ⚠️ The runner is **sqlite-only** — see `references/cli.md`.

## File uploads (storage)

1. `pola generate storage --driver fs --root uploads` creates `StorageBlob` (key, filename,
   content_type, byte_size, checksum) and `StorageAttachment` (polymorphic blob↔record join), plus
   their repositories, and adds the `storage` block to `Polafile.hcl`.
2. Attach to a model with a blob reference: `pola generate scaffold Document title:string
   file:references{StorageBlob}`. A `references{StorageBlob}` field marks the resource as having a
   file upload, so the generated route handles multipart form data for that field.
3. With `image_processing` enabled, the `/_pola/image` endpoint and the `ImageProcessing.processURL` bridge binding can resize/crop
   uploaded images on the fly.
