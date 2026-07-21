package gorm

import (
	"context"
	"testing"
	"time"

	"github.com/polagonow/pola/repository"
)

// Stamped carries the framework-managed lifecycle timestamps.
type Stamped struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func TestWithClockStampsDeterministically(t *testing.T) {
	db := openDB(t, &Stamped{})
	fixed := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := New[Stamped, uint](db, repository.WithClock[uint](func() time.Time { return fixed }))
	ctx := context.Background()

	e := &Stamped{Name: "x"}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !e.CreatedAt.Equal(fixed) || !e.UpdatedAt.Equal(fixed) {
		t.Errorf("Create stamps = created %v / updated %v, want both %v", e.CreatedAt, e.UpdatedAt, fixed)
	}

	later := fixed.Add(time.Hour)
	repo2 := New[Stamped, uint](db, repository.WithClock[uint](func() time.Time { return later }))
	e.Name = "y"
	if err := repo2.Update(ctx, e); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !e.UpdatedAt.Equal(later) {
		t.Errorf("Update stamp = %v, want %v", e.UpdatedAt, later)
	}
	if !e.CreatedAt.Equal(fixed) {
		t.Errorf("CreatedAt changed on update: %v, want %v", e.CreatedAt, fixed)
	}
}

func TestDefaultClockIsTimeNow(t *testing.T) {
	db := openDB(t, &Stamped{})
	repo := New[Stamped, uint](db) // no WithClock → defaults to time.Now
	e := &Stamped{Name: "x"}
	before := time.Now().Add(-time.Minute)
	if err := repo.Create(context.Background(), e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.CreatedAt.Before(before) || e.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("default clock CreatedAt = %v, want ~now", e.CreatedAt)
	}
}
