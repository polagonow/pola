package repository

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Operator is a filter comparison. The set mirrors the widely-used Goyave
// `filter` vocabulary so query strings are familiar and portable.
type Operator string

const (
	OpEq      Operator = "$eq"      // column = value
	OpNe      Operator = "$ne"      // column <> value
	OpGt      Operator = "$gt"      // column > value
	OpLt      Operator = "$lt"      // column < value
	OpGte     Operator = "$gte"     // column >= value
	OpLte     Operator = "$lte"     // column <= value
	OpCont    Operator = "$cont"    // column LIKE %value%
	OpStarts  Operator = "$starts"  // column LIKE value%
	OpEnds    Operator = "$ends"    // column LIKE %value
	OpIn      Operator = "$in"      // column IN (values...)
	OpNotIn   Operator = "$notin"   // column NOT IN (values...)
	OpBetween Operator = "$between" // column BETWEEN a AND b
	OpIsNull  Operator = "$isnull"  // column IS NULL
	OpNotNull Operator = "$notnull" // column IS NOT NULL
)

// validOperators is the recognized set; unknown operators are dropped by the
// parser so a malformed query never reaches the adapter.
var validOperators = map[Operator]bool{
	OpEq: true, OpNe: true, OpGt: true, OpLt: true, OpGte: true, OpLte: true,
	OpCont: true, OpStarts: true, OpEnds: true, OpIn: true, OpNotIn: true,
	OpBetween: true, OpIsNull: true, OpNotNull: true,
}

// Filter is a single WHERE condition against a column. Args carries the
// operator's operands: none for $isnull/$notnull, two for $between, the list for
// $in/$notin, and one otherwise.
type Filter struct {
	Field    string   `json:"field"`
	Operator Operator `json:"operator"`
	Args     []string `json:"args,omitempty"`
}

// Sort is a single ORDER BY clause.
type Sort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

// ParseListQuery builds a ListParams from URL query values using a compact,
// Goyave-compatible syntax. It is transport-agnostic — pass
// request.URL.Query() from any web adapter:
//
//	?page=2&per_page=20
//	&filter=name||$cont||jack        (repeatable; ANDed)
//	&filter=age||$between||18,30
//	&filter=role||$in||admin,staff
//	&sort=created_at,desc            (repeatable; applied in order)
//	&fields=id,name,email
//
// Parsing is permissive about values but strict about structure: filters with an
// unknown operator or missing operands, and empty field names, are skipped
// rather than erroring, so a hostile or sloppy query degrades to fewer
// constraints instead of a 500. Column names are NOT validated here (the parser
// doesn't know the entity); the adapter enforces the per-entity allowlist.
func ParseListQuery(v url.Values) ListParams {
	var p ListParams
	p.Page, _ = strconv.Atoi(v.Get("page"))
	p.PerPage, _ = strconv.Atoi(v.Get("per_page"))

	for _, raw := range v["filter"] {
		if f, ok := parseFilter(raw); ok {
			p.Filters = append(p.Filters, f)
		}
	}
	for _, raw := range v["sort"] {
		field, dir, _ := strings.Cut(raw, ",")
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		p.Sorts = append(p.Sorts, Sort{Field: field, Desc: strings.EqualFold(strings.TrimSpace(dir), "desc")})
	}
	if fields := v.Get("fields"); fields != "" {
		for _, f := range strings.Split(fields, ",") {
			if f = strings.TrimSpace(f); f != "" {
				p.Fields = append(p.Fields, f)
			}
		}
	}
	return p
}

// parseFilter decodes one "field||$op||value" triple. The value segment is
// optional for the null operators and comma-split for the multi-arg ones.
func parseFilter(raw string) (Filter, bool) {
	parts := strings.SplitN(raw, "||", 3)
	if len(parts) < 2 {
		return Filter{}, false
	}
	field := strings.TrimSpace(parts[0])
	op := Operator(strings.TrimSpace(parts[1]))
	if field == "" || !validOperators[op] {
		return Filter{}, false
	}

	f := Filter{Field: field, Operator: op}
	switch op {
	case OpIsNull, OpNotNull:
		return f, true // no operands
	case OpIn, OpNotIn, OpBetween:
		if len(parts) < 3 {
			return Filter{}, false
		}
		for _, a := range strings.Split(parts[2], ",") {
			f.Args = append(f.Args, strings.TrimSpace(a))
		}
		if op == OpBetween && len(f.Args) != 2 {
			return Filter{}, false
		}
		if len(f.Args) == 0 {
			return Filter{}, false
		}
	default:
		if len(parts) < 3 {
			return Filter{}, false
		}
		f.Args = []string{parts[2]}
	}
	return f, true
}

// Coerce converts a string operand to the Go type of its column so comparisons
// bind with the right type on strict drivers (e.g. Postgres won't compare an
// integer column against a text parameter). Unknown/complex types fall back to
// the raw string.
func Coerce(t reflect.Type, s string) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			return n
		}
	case reflect.Float32, reflect.Float64:
		if fl, err := strconv.ParseFloat(s, 64); err == nil {
			return fl
		}
	}
	return s
}
