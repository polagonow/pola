package gorm

import (
	"context"
	"errors"
	"testing"
)

func TestTransactionCommits(t *testing.T) {
	db := openDB(t, &TestWidget{})
	mgr := NewTxManager(db)
	repo := New[TestWidget, uint](db)
	ctx := context.Background()

	err := mgr.Transaction(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, &TestWidget{Name: "a"}); err != nil {
			return err
		}
		return repo.Create(ctx, &TestWidget{Name: "b"})
	})
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}

	var n int64
	db.Model(&TestWidget{}).Count(&n)
	if n != 2 {
		t.Errorf("committed rows = %d, want 2", n)
	}
}

func TestTransactionRollsBackOnError(t *testing.T) {
	db := openDB(t, &TestWidget{})
	mgr := NewTxManager(db)
	repo := New[TestWidget, uint](db)
	ctx := context.Background()

	boom := errors.New("boom")
	err := mgr.Transaction(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, &TestWidget{Name: "a"}); err != nil {
			return err
		}
		return boom // should roll the whole unit back
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Transaction err = %v, want boom", err)
	}

	var n int64
	db.Model(&TestWidget{}).Count(&n)
	if n != 0 {
		t.Errorf("rows after rollback = %d, want 0", n)
	}
}

func TestNestedTransactionSavepoint(t *testing.T) {
	db := openDB(t, &TestWidget{})
	mgr := NewTxManager(db)
	repo := New[TestWidget, uint](db)
	ctx := context.Background()

	err := mgr.Transaction(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, &TestWidget{Name: "outer"}); err != nil {
			return err
		}
		// Inner unit fails and is swallowed: it must roll back to its savepoint
		// without discarding the outer transaction.
		_ = mgr.Transaction(ctx, func(ctx context.Context) error {
			if err := repo.Create(ctx, &TestWidget{Name: "inner"}); err != nil {
				return err
			}
			return errors.New("inner boom")
		})
		return nil // commit the outer unit
	})
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}

	var names []string
	db.Model(&TestWidget{}).Order("id").Pluck("name", &names)
	if len(names) != 1 || names[0] != "outer" {
		t.Errorf("rows = %v, want [outer] (inner rolled back to savepoint)", names)
	}
}

func TestDBFallsBackWithoutTransaction(t *testing.T) {
	db := openDB(t, &TestWidget{})
	if got := DB(context.Background(), db); got != db {
		t.Error("DB should return the fallback handle when no transaction is active")
	}
}
