package repositories

const DefaultPerPage = 25

type ListParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func (p ListParams) Normalize() ListParams {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = DefaultPerPage
	}
	return p
}

func (p ListParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

type ListResult[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}
