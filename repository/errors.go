package repository

import "errors"

// ErrNotFound is the ORM-neutral sentinel returned by Get when no row matches.
// Each adapter maps its own driver error (gorm.ErrRecordNotFound,
// beego orm.ErrNoRows, ent's *NotFoundError) onto it, so callers can branch on
// "not found" — e.g. to answer HTTP 404 — without importing an ORM package:
//
//	u, err := repo.Get(ctx, id)
//	if repository.IsNotFound(err) {
//	    return response.NoContent(http.StatusNotFound)
//	}
//
// Its message ("record not found") matches gorm's, so wrapped error strings are
// unchanged for the gorm engine.
var ErrNotFound = errors.New("record not found")

// IsNotFound reports whether err is (or wraps) [ErrNotFound]. It is shorthand
// for errors.Is(err, ErrNotFound).
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
