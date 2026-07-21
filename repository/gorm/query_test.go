package gorm

import (
	"context"
	"net/url"
	"testing"

	"github.com/polagonow/pola/repository"
)

type Product struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	Price  int
	Tag    string
	Secret string
}

func seedProducts(t *testing.T) repository.Repository[Product, uint] {
	t.Helper()
	db := openDB(t, &Product{})
	repo := New[Product, uint](db)
	ctx := context.Background()
	for _, p := range []Product{
		{Name: "Apple", Price: 100, Tag: "fruit", Secret: "s1"},
		{Name: "Apricot", Price: 150, Tag: "fruit", Secret: "s2"},
		{Name: "Banana", Price: 50, Tag: "fruit", Secret: "s3"},
		{Name: "Carrot", Price: 80, Tag: "veg", Secret: "s4"},
	} {
		p := p
		if err := repo.Create(ctx, &p); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return repo
}

func list(t *testing.T, repo repository.Repository[Product, uint], p repository.ListParams) *repository.ListResult[*Product] {
	t.Helper()
	res, err := repo.List(context.Background(), p)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return res
}

func TestFilterEq(t *testing.T) {
	repo := seedProducts(t)
	res := list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "tag", Operator: repository.OpEq, Args: []string{"veg"}}},
	})
	if res.Total != 1 || len(res.Items) != 1 || res.Items[0].Name != "Carrot" {
		t.Errorf("eq tag=veg → total %d items %d, want the single Carrot", res.Total, len(res.Items))
	}
}

func TestFilterMalformedArgsDoNotPanic(t *testing.T) {
	repo := seedProducts(t)
	// A Filter can be constructed directly (bypassing ParseListQuery's arity
	// validation), so the adapter must not index missing operands. These would
	// otherwise panic ($between) or emit invalid SQL ($in).
	cases := map[string]repository.Filter{
		"between one arg": {Field: "price", Operator: repository.OpBetween, Args: []string{"5"}},
		"between no args": {Field: "price", Operator: repository.OpBetween},
		"in no args":      {Field: "tag", Operator: repository.OpIn},
		"notin no args":   {Field: "tag", Operator: repository.OpNotIn},
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			res := list(t, repo, repository.ListParams{Filters: []repository.Filter{f}})
			// Malformed filter is skipped, so every seeded row is returned.
			if res.Total != 4 {
				t.Errorf("malformed filter should be dropped; total = %d, want 4", res.Total)
			}
		})
	}
}

func TestFilterContAndLikeEscaping(t *testing.T) {
	repo := seedProducts(t)

	res := list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "name", Operator: repository.OpCont, Args: []string{"an"}}},
	})
	if res.Total != 1 || res.Items[0].Name != "Banana" {
		t.Errorf("cont 'an' → %+v, want just Banana", names(res))
	}

	// A literal "%" must be escaped: without escaping it would match every row.
	res = list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "name", Operator: repository.OpCont, Args: []string{"%"}}},
	})
	if res.Total != 0 {
		t.Errorf("cont '%%' → total %d, want 0 (wildcard escaped)", res.Total)
	}
}

func TestFilterInAndBetween(t *testing.T) {
	repo := seedProducts(t)

	in := list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "price", Operator: repository.OpIn, Args: []string{"50", "80"}}},
	})
	if in.Total != 2 {
		t.Errorf("price in (50,80) → total %d, want 2", in.Total)
	}

	between := list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "price", Operator: repository.OpBetween, Args: []string{"60", "120"}}},
	})
	if between.Total != 2 {
		t.Errorf("price between 60..120 → total %d, want 2 (Apple, Carrot)", between.Total)
	}
}

func TestSortAndFieldSelect(t *testing.T) {
	repo := seedProducts(t)

	sorted := list(t, repo, repository.ListParams{
		Sorts: []repository.Sort{{Field: "price", Desc: true}},
	})
	if sorted.Items[0].Price != 150 {
		t.Errorf("first by price desc = %d, want 150", sorted.Items[0].Price)
	}

	// Selecting only "name" leaves other columns at their zero value.
	selected := list(t, repo, repository.ListParams{
		Fields: []string{"name"},
		Sorts:  []repository.Sort{{Field: "name"}},
	})
	if selected.Items[0].Name == "" || selected.Items[0].Price != 0 {
		t.Errorf("field-select name → %+v, want Name set and Price zeroed", selected.Items[0])
	}
}

func TestUnknownAndBlacklistedColumnsIgnored(t *testing.T) {
	// Unknown column: filter is dropped, so all rows come back.
	repo := seedProducts(t)
	res := list(t, repo, repository.ListParams{
		Filters: []repository.Filter{{Field: "bogus", Operator: repository.OpEq, Args: []string{"x"}}},
	})
	if res.Total != 4 {
		t.Errorf("unknown-column filter → total %d, want 4 (dropped)", res.Total)
	}

	// Blacklisted column: even though "secret" exists, filtering by it is refused.
	db := openDB(t, &Product{})
	guarded := New[Product, uint](db, repository.WithBlacklist[uint]("secret"))
	ctx := context.Background()
	_ = guarded.Create(ctx, &Product{Name: "X", Secret: "top"})
	got := list(t, guarded, repository.ListParams{
		Filters: []repository.Filter{{Field: "secret", Operator: repository.OpEq, Args: []string{"top"}}},
	})
	if got.Total != 1 {
		t.Errorf("blacklisted filter should be dropped (total %d, want 1 unfiltered)", got.Total)
	}
}

func TestListParamsFromQueryString(t *testing.T) {
	repo := seedProducts(t)
	v, _ := url.ParseQuery("filter=tag||$eq||fruit&sort=price,desc&per_page=2")
	res := list(t, repo, repository.ParseListQuery(v))
	if res.Total != 3 {
		t.Errorf("tag=fruit → total %d, want 3", res.Total)
	}
	if len(res.Items) != 2 {
		t.Errorf("per_page=2 → items %d, want 2", len(res.Items))
	}
	if res.Items[0].Price != 150 {
		t.Errorf("sorted desc → first price %d, want 150", res.Items[0].Price)
	}
}

func names(res *repository.ListResult[*Product]) []string {
	out := make([]string, len(res.Items))
	for i, p := range res.Items {
		out[i] = p.Name
	}
	return out
}
