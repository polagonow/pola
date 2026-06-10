package repositories

// ListParams holds pagination parameters.
type ListParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
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
