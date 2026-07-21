# uow — unit of work

Framework-neutral **business transactions**. A service wraps the mutating steps
of an operation in `TxManager.Transaction`; repositories enlist automatically, so
a single business action spanning several repositories either fully commits or
fully rolls back.

```go
type TxManager interface {
    Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

## Why

`repository.Repository` methods are atomic individually, but a service that
writes two repositories had no way to make the pair atomic. `uow.TxManager`
fills that gap **without** leaking a database handle into the domain layer: the
transaction travels in the `context.Context`, and repositories resolve it
transparently.

## How it fits together

The neutral interface lives here; the concrete manager lives beside its ORM
adapter. For GORM:

```go
mgr  := gorm.NewTxManager(db)          // implements uow.TxManager
repo := gorm.New[models.Account, int64](db)

// In a service constructed with (mgr, repo):
func (s *BillingService) Transfer(ctx context.Context, from, to int64, cents int64) error {
    return s.tx.Transaction(ctx, func(ctx context.Context) error {
        if err := s.accounts.Debit(ctx, from, cents); err != nil {
            return err // rolls back
        }
        return s.accounts.Credit(ctx, to, cents) // commits when nil
    })
}
```

Inside `gorm`, every repository method routes its query through
`gorm.DB(ctx, r.db)`, which returns the transaction stashed in `ctx` (or the
plain handle when none is active) — so the same `Debit`/`Credit` code works both
inside and outside a transaction, with no branching.

- **Commit/rollback** is automatic: return `nil` to commit, an error to roll back.
- **Nesting** is safe: a `Transaction` call already inside a transaction opens a
  SAVEPOINT, so a failing inner unit rolls back to its start without discarding
  the outer one.
