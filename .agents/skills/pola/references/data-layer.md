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
`actions/` (React bridge) **or** `routes/` (HTTP). `pola generate scaffold` produces all of it at
once; the individual generators produce one layer each. Below is the verified shape of each layer
(gorm ORM).

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
Shared pagination (`repositories/pagination.go`, written once per project):
```go
type ListParams struct {
    Page    int `json:"page"`
    PerPage int `json:"per_page"`
}
const DefaultPerPage = 25
func (p ListParams) Normalize() ListParams { /* clamps Page>=1, PerPage>=DefaultPerPage */ }
func (p ListParams) Offset() int           { return (p.Page - 1) * p.PerPage }

type ListResult[T any] struct {
    Items      []T `json:"items"`
    Total      int `json:"total"`
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    TotalPages int `json:"total_pages"`
}
```

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

## ORM selection

`database.orm` in `Polafile.hcl` picks the templates: `gorm` (models under `db/models/gorm/`,
embeds `gorm.Model`) or `ent` (schemas under `db/models/schema/`). The repository implementation
package (`repositories/gorm/` vs `repositories/ent/`) follows the same choice. Change ORM by editing
the Polafile before generating.

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
