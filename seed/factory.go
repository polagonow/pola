package seed

import "context"

// Generator builds one populated instance of T — typically with random data
// from a faker library of your choice (the factory stays dependency-free by
// letting you supply the generator):
//
//	var userFactory = seed.NewFactory(func() *model.User {
//	    return &model.User{Name: faker.Name(), Email: faker.Email()}
//	})
type Generator[T any] func() *T

// Factory produces test/seed records for T from a [Generator], optionally
// applying a field override to every record. It mirrors the ergonomics of
// Goyave's database factories: generate in memory, or generate-and-persist.
type Factory[T any] struct {
	gen      Generator[T]
	override func(*T)
}

// NewFactory returns a Factory driven by gen.
func NewFactory[T any](gen Generator[T]) *Factory[T] { return &Factory[T]{gen: gen} }

// Override registers a mutation applied to every generated record after the
// generator runs — e.g. to pin a field across a whole batch. It returns the
// factory for chaining and replaces any previous override.
//
//	userFactory.Override(func(u *model.User) { u.TeamID = team.ID }).Save(ctx, repo.Create, 10)
func (f *Factory[T]) Override(fn func(*T)) *Factory[T] {
	f.override = fn
	return f
}

// Generate returns n freshly generated records (not persisted). A non-positive n
// yields an empty slice.
func (f *Factory[T]) Generate(n int) []*T {
	if n < 0 {
		n = 0
	}
	out := make([]*T, 0, n)
	for i := 0; i < n; i++ {
		e := f.gen()
		if e != nil && f.override != nil {
			f.override(e)
		}
		out = append(out, e)
	}
	return out
}

// Save generates n records and persists each via create — typically a
// repository's Create method (`repo.Create`) — returning the records (with any
// IDs the create call populated). It stops at and returns the first error along
// with the records generated so far.
func (f *Factory[T]) Save(ctx context.Context, create func(context.Context, *T) error, n int) ([]*T, error) {
	items := f.Generate(n)
	for _, e := range items {
		if err := create(ctx, e); err != nil {
			return items, err
		}
	}
	return items, nil
}
