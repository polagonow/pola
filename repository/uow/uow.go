// Package uow defines the framework-neutral business-transaction ("unit of
// work") contract that lets a service run several repository operations
// atomically without depending on any ORM.
//
// A service takes a [TxManager] as a constructor dependency and wraps the
// mutating steps of a business operation in Transaction. The manager stashes the
// live database handle in the context.Context it passes to the callback;
// repositories resolve that handle transparently (see the ORM-specific
// implementation, e.g. repository/gorm.DB), so the same repository method works
// inside or outside a transaction with no branching.
//
//	func (s *BillingService) Transfer(ctx context.Context, from, to int64, cents int64) error {
//	    return s.tx.Transaction(ctx, func(ctx context.Context) error {
//	        if err := s.accounts.Debit(ctx, from, cents); err != nil {
//	            return err // rolls back
//	        }
//	        return s.accounts.Credit(ctx, to, cents) // commits if nil
//	    })
//	}
//
// The neutral interface lives here (importable without pulling in a database
// driver); the concrete manager lives beside its ORM adapter.
package uow

import "context"

// TxManager runs a function within a business transaction.
//
// The callback receives a context carrying the active transaction. Returning a
// non-nil error rolls the transaction back; returning nil commits it. Calling
// Transaction from within an active transaction nests it (implementations back
// nesting with savepoints) so a failing inner unit rolls back to its start
// without discarding the outer one.
type TxManager interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
