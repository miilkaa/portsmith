// Package pagination provides offset-based and cursor-based pagination primitives.
package pagination

import (
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// OffsetPage represents offset-based pagination parameters.
type OffsetPage struct {
	page  int
	limit int
}

// NewOffsetPage creates an OffsetPage with bounds checking.
func NewOffsetPage(page, limit int) OffsetPage {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return OffsetPage{page: page, limit: limit}
}

// OffsetFromQuery parses pagination from HTTP query parameters.
// Reads ?page=1&limit=20 with safe defaults and capped limit.
func OffsetFromQuery(r *http.Request) OffsetPage {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	return NewOffsetPage(page, limit)
}

// Offset returns the SQL OFFSET value.
func (p OffsetPage) Offset() int {
	return (p.page - 1) * p.limit
}

// Limit returns the SQL LIMIT value.
func (p OffsetPage) Limit() int {
	return p.limit
}

// PageNumber returns the current page number (1-based).
func (p OffsetPage) PageNumber() int {
	return p.page
}
