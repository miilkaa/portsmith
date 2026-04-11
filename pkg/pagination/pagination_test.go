package pagination_test

// pagination_test.go — контрактные тесты для pkg/pagination.
//
// Контракт:
//  1. NewOffsetPage применяет bounds: page>=1, 1<=limit<=100.
//  2. Offset() возвращает корректное смещение.
//  3. OffsetFromQuery парсит query-строку (?page=&limit=).
//  4. TotalPages вычисляет количество страниц.

import (
	"net/http"
	"testing"

	"github.com/miilkaa/portsmith/pkg/pagination"
)

func TestNewOffsetPage_defaults(t *testing.T) {
	p := pagination.NewOffsetPage(0, 0)
	if p.PageNumber() != 1 {
		t.Errorf("expected page 1, got %d", p.PageNumber())
	}
	if p.Limit() != 20 {
		t.Errorf("expected limit 20, got %d", p.Limit())
	}
}

func TestNewOffsetPage_clampMaxLimit(t *testing.T) {
	p := pagination.NewOffsetPage(1, 9999)
	if p.Limit() != 100 {
		t.Errorf("expected capped limit 100, got %d", p.Limit())
	}
}

func TestOffsetPage_offset(t *testing.T) {
	cases := []struct {
		page, limit, wantOffset int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 25, 50},
	}
	for _, tc := range cases {
		p := pagination.NewOffsetPage(tc.page, tc.limit)
		if got := p.Offset(); got != tc.wantOffset {
			t.Errorf("page=%d limit=%d: expected offset %d, got %d", tc.page, tc.limit, tc.wantOffset, got)
		}
	}
}

func TestOffsetFromQuery_parseQueryString(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/?page=3&limit=15", nil)
	p := pagination.OffsetFromQuery(req)

	if p.PageNumber() != 3 {
		t.Errorf("expected page 3, got %d", p.PageNumber())
	}
	if p.Limit() != 15 {
		t.Errorf("expected limit 15, got %d", p.Limit())
	}
	if p.Offset() != 30 {
		t.Errorf("expected offset 30, got %d", p.Offset())
	}
}

func TestOffsetFromQuery_missingParams_usesDefaults(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	p := pagination.OffsetFromQuery(req)

	if p.PageNumber() != 1 {
		t.Errorf("expected page 1, got %d", p.PageNumber())
	}
	if p.Limit() != 20 {
		t.Errorf("expected limit 20, got %d", p.Limit())
	}
}

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total int64
		limit int
		want  int
	}{
		{0, 10, 0},
		{10, 10, 1},
		{11, 10, 2},
		{100, 20, 5},
		{101, 20, 6},
	}
	for _, tc := range cases {
		got := pagination.TotalPages(tc.total, tc.limit)
		if got != tc.want {
			t.Errorf("total=%d limit=%d: expected %d pages, got %d", tc.total, tc.limit, tc.want, got)
		}
	}
}
