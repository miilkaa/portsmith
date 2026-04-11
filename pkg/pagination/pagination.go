// Package pagination provides offset-based pagination primitives for
// HTTP handlers and GORM repositories.
//
// Usage in handler:
//
//	page := pagination.OffsetFromQuery(r)
//	users, total, _ := svc.List(ctx, filter, page)
//	totalPages := pagination.TotalPages(total, page.Limit())
//
// Usage in repository:
//
//	query.Offset(page.Offset()).Limit(page.Limit())
package pagination

import (
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// OffsetPage holds validated offset-based pagination parameters.
type OffsetPage struct {
	page  int
	limit int
}

// NewOffsetPage creates an OffsetPage with safe bounds:
// page is clamped to [1, ∞), limit is clamped to [1, 100].
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

// OffsetFromQuery parses ?page= and ?limit= from the HTTP request.
// Missing or invalid values fall back to safe defaults.
func OffsetFromQuery(r *http.Request) OffsetPage {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	return NewOffsetPage(page, limit)
}

// Offset returns the SQL OFFSET value: (page-1) * limit.
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

// TotalPages calculates the total number of pages given total record count and page size.
func TotalPages(total int64, limit int) int {
	if total == 0 || limit <= 0 {
		return 0
	}
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}
	return pages
}
