package gorm

import (
	"context"

	"gorm.io/gorm"

	"github.com/polagonow/pola/repository/uow"
)

// txKey is the private context key under which a running transaction's *gorm.DB
// is stored. Keeping it unexported means the only way to enlist in a transaction
// is through TxManager, and the only way to read it is DB.
type txKey struct{}

// DB returns the transaction handle stored in ctx by [TxManager], or fallback
// when no transaction is active. Every generic repository method routes its
// queries through DB(ctx, r.db), so the same call joins an ambient transaction
// when one exists and runs standalone otherwise — no conditional logic in the
// repository.
func DB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback
}

// TxManager is the GORM implementation of [uow.TxManager]. It propagates the
// transaction via context so repositories enlist transparently.
type TxManager struct {
	db *gorm.DB
}

// NewTxManager returns a TxManager bound to db. Register it in the DI container
// and inject it into services that need atomic multi-repository operations.
func NewTxManager(db *gorm.DB) *TxManager { return &TxManager{db: db} }

var _ uow.TxManager = (*TxManager)(nil)

// Transaction runs fn inside a GORM transaction, committing when fn returns nil
// and rolling back otherwise. It resolves the starting handle with DB, so a
// Transaction call already inside a transaction opens a SAVEPOINT (nested unit)
// rather than a second top-level transaction.
func (m *TxManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return DB(ctx, m.db).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
