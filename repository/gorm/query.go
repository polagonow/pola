package gorm

import (
	"reflect"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/sqlutil"
)

// filterableColumns returns the real database columns of entity T keyed by the
// exact column name GORM uses (via its naming strategy, so acronyms like UserID
// → user_id match what queries emit), mapped to the Go field type for operand
// coercion. Association fields (no column) are excluded, so query strings can
// only ever reach real columns. On a schema-parse failure it returns an empty
// map, disabling filtering rather than risking a bad column reference.
func filterableColumns[T any](db *gorm.DB) map[string]reflect.Type {
	sch, err := schema.Parse(new(T), &sync.Map{}, db.NamingStrategy)
	if err != nil {
		return map[string]reflect.Type{}
	}
	out := make(map[string]reflect.Type, len(sch.Fields))
	for _, f := range sch.Fields {
		if f.DBName == "" {
			continue // association / non-column field
		}
		out[f.DBName] = f.FieldType
	}
	return out
}

// allowed reports whether col is a real, non-blacklisted column of T, returning
// its field type for operand coercion.
func (r *repo[T, ID]) allowed(col string) (reflect.Type, bool) {
	if r.blacklist[col] {
		return nil, false
	}
	t, ok := r.columns[col]
	return t, ok
}

// filterScope returns a GORM scope applying params.Filters (ANDed) against the
// column allowlist. It is shared by the Count and Find queries so a paginated
// total reflects the same filters as the returned page.
func (r *repo[T, ID]) filterScope(params repository.ListParams) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for _, f := range params.Filters {
			typ, ok := r.allowed(f.Field)
			if !ok {
				continue // unknown or blacklisted column: silently dropped
			}
			db = applyFilter(db, f, typ)
		}
		return db
	}
}

// applyFilter adds one condition. f.Field is an allowlisted column name, so
// interpolating it into the fragment is safe; every operand is bound as a
// parameter (and LIKE operands are wildcard-escaped).
func applyFilter(db *gorm.DB, f repository.Filter, typ reflect.Type) *gorm.DB {
	col := f.Field
	switch f.Operator {
	case repository.OpEq:
		return db.Where(col+" = ?", repository.Coerce(typ, arg0(f)))
	case repository.OpNe:
		return db.Where(col+" <> ?", repository.Coerce(typ, arg0(f)))
	case repository.OpGt:
		return db.Where(col+" > ?", repository.Coerce(typ, arg0(f)))
	case repository.OpLt:
		return db.Where(col+" < ?", repository.Coerce(typ, arg0(f)))
	case repository.OpGte:
		return db.Where(col+" >= ?", repository.Coerce(typ, arg0(f)))
	case repository.OpLte:
		return db.Where(col+" <= ?", repository.Coerce(typ, arg0(f)))
	case repository.OpCont:
		return db.Where(col+" LIKE ? ESCAPE '\\'", "%"+sqlutil.EscapeLike(arg0(f))+"%")
	case repository.OpStarts:
		return db.Where(col+" LIKE ? ESCAPE '\\'", sqlutil.EscapeLike(arg0(f))+"%")
	case repository.OpEnds:
		return db.Where(col+" LIKE ? ESCAPE '\\'", "%"+sqlutil.EscapeLike(arg0(f)))
	case repository.OpIn:
		return db.Where(col+" IN ?", coerceAll(typ, f.Args))
	case repository.OpNotIn:
		return db.Where(col+" NOT IN ?", coerceAll(typ, f.Args))
	case repository.OpBetween:
		return db.Where(col+" BETWEEN ? AND ?", repository.Coerce(typ, f.Args[0]), repository.Coerce(typ, f.Args[1]))
	case repository.OpIsNull:
		return db.Where(col + " IS NULL")
	case repository.OpNotNull:
		return db.Where(col + " IS NOT NULL")
	}
	return db
}

func arg0(f repository.Filter) string {
	if len(f.Args) > 0 {
		return f.Args[0]
	}
	return ""
}

func coerceAll(typ reflect.Type, args []string) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = repository.Coerce(typ, a)
	}
	return out
}

// selectColumns returns the allowlisted subset of params.Fields, or nil when no
// valid selection was requested (i.e. SELECT *).
func (r *repo[T, ID]) selectColumns(params repository.ListParams) []string {
	var cols []string
	for _, c := range params.Fields {
		if _, ok := r.allowed(c); ok {
			cols = append(cols, c)
		}
	}
	return cols
}

// orderScope applies params.Sorts (allowlisted) in request order.
func (r *repo[T, ID]) orderScope(db *gorm.DB, params repository.ListParams) *gorm.DB {
	for _, s := range params.Sorts {
		if _, ok := r.allowed(s.Field); !ok {
			continue
		}
		dir := "ASC"
		if s.Desc {
			dir = "DESC"
		}
		db = db.Order(s.Field + " " + dir)
	}
	return db
}
