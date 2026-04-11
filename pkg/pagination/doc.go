// Package pagination provides offset-based pagination for HTTP handlers and repositories.
//
// # In the handler
//
//	page := pagination.OffsetFromQuery(c.Request)  // parses ?page=&limit=
//	users, total, _ := svc.List(ctx, filter, page)
//	totalPages := pagination.TotalPages(total, page.Limit())
//
// # In the repository
//
//	query.Offset(page.Offset()).Limit(page.Limit())
//
// # Defaults and limits
//
//   - Default page: 1
//   - Default limit: 20
//   - Maximum limit: 100 (capped automatically)
package pagination
