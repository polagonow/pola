package repositories

import "github.com/polagonow/pola/repository"

// ListParams holds pagination parameters. It aliases the framework type so
// generated services, routes and hand-written code share one definition with
// the framework's generic repositories.
type ListParams = repository.ListParams

// ListResult wraps a page of results with pagination metadata.
type ListResult[T any] = repository.ListResult[T]

// DefaultPerPage is the default number of items per page.
const DefaultPerPage = repository.DefaultPerPage
