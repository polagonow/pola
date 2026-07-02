# Service

Framework-owned, generic business-logic layer — the service-layer twin of
`repository`. One neutral CRUD contract that wraps a `repository.Repository`,
with a generated per-entity wrapper as the place to add business rules.

```go
// The contract every generated service interface embeds.
type Service[T any, ID comparable] interface {
    Create(ctx context.Context, entity *T) error
    Get(ctx context.Context, id ID) (*T, error)
    List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*T], error)
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id ID) error
}
```

## Why

A generated service used to be ~57 lines of pure pass-through to the repository
(plus a ~150-line delegation test) that varied only by entity name and ID type.
That delegation now lives in the framework once, and generated code shrinks to a
slim wrapper embedding the generic service:

```go
// services/user_service.go (generated)
type UserServiceInterface interface {
    service.Service[repositories.User, uint] // add custom business methods here
}

type UserService struct {
    service.Service[repositories.User, uint]
    repo repositories.UserRepository // retained for custom methods / overrides
}

func NewUserService(repo repositories.UserRepository) *UserService {
    return &UserService{Service: service.New[repositories.User, uint](repo), repo: repo}
}

var _ UserServiceInterface = (*UserService)(nil)
```

Routes and other callers depend on the named `UserServiceInterface`, so the
embedding is invisible to them; DI wiring (`NewUserService(repo)`) is unchanged.

## Adding business logic

By default each method delegates straight to the repository. To add validation
or business rules, **override the method** on the generated struct — the
override shadows the embedded generic method, and you have `s.repo` available:

```go
func (s *UserService) Create(ctx context.Context, u *repositories.User) error {
    if u.Email == "" {
        return errors.New("email required")
    }
    return s.repo.Create(ctx, u)
}
```

Custom queries or cross-entity logic go on the same struct, using `s.repo` (or
additional repositories injected via the constructor).

## Relationship to `repository`

`Service[T, ID]` is structurally identical to `repository.Repository[T, ID]`
but kept a distinct named type on purpose: it preserves the architectural seam
where business logic belongs, independent of persistence. `service.New` accepts
any `repository.Repository[T, ID]`, so a generated `repositories.XRepository`
(which embeds it) drops straight in.
