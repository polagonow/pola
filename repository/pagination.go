package repository

// ListParams holds pagination parameters plus optional query-driven filtering,
// sorting and field selection. The three query fields are additive and
// backward-compatible: a zero-value ListParams (or one built only with
// Page/PerPage) behaves exactly as plain pagination. Adapters that understand
// them (currently gorm) apply Filters/Sorts/Fields against an allowlist of the
// entity's columns; adapters that don't simply ignore them.
type ListParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`

	// Filters are WHERE conditions, ANDed together.
	Filters []Filter `json:"filters,omitempty"`
	// Sorts are ORDER BY clauses, applied in order.
	Sorts []Sort `json:"sorts,omitempty"`
	// Fields is an optional SELECT allowlist (empty means all columns).
	Fields []string `json:"fields,omitempty"`
}

// DefaultPerPage is the default number of items per page.
const DefaultPerPage = 25

// Normalize sets defaults for zero-value fields.
func (p ListParams) Normalize() ListParams {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = DefaultPerPage
	}
	return p
}

// Offset returns the SQL offset for the current page.
func (p ListParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// ListResult wraps a page of results with pagination metadata.
type ListResult[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// NewListResult assembles a ListResult from a page of items, the total row
// count and the (already normalized) params. TotalPages uses ceiling
// division, matching the arithmetic previously inlined in every generated
// repository.
func NewListResult[T any](items []T, total int, params ListParams) *ListResult[T] {
	totalPages := total / params.PerPage
	if total%params.PerPage != 0 {
		totalPages++
	}
	return &ListResult[T]{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}
}
