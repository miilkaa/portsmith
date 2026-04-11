package gen_test

// gen_test.go — контрактные тесты для internal/gen.
//
// Контракт генератора:
//  1. CollectRepoCalls находит все методы репозитория, вызываемые в коде.
//  2. CollectServiceCalls находит все методы сервиса, вызываемые в коде.
//  3. MethodSigs парсит AST файла и возвращает сигнатуры методов.
//  4. PortPrefix строит правильный префикс из имени директории.
//  5. DetectModulePath читает module из go.mod в заданном корне.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miilkaa/portsmith/internal/gen"
)

func TestCollectRepoCalls_directAndAlias(t *testing.T) {
	src := `
func (s *Service) DoWork(ctx context.Context) error {
	s.repo.DirectCall(ctx)
	r := s.repo
	r.AliasCall(ctx)
	return nil
}`
	got := gen.CollectRepoCalls(src)
	for _, want := range []string{"DirectCall", "AliasCall"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing repo method %q in %v", want, got)
		}
	}
}

func TestCollectRepoCalls_skipFalsePositive(t *testing.T) {
	src := `stats, err := h.repository.GetUsageStats(ctx, nil)`
	got := gen.CollectRepoCalls(src)
	if _, ok := got["Error"]; ok {
		t.Error("must not treat err := ... as repo alias")
	}
	if _, ok := got["GetUsageStats"]; !ok {
		t.Error("direct h.repository.GetUsageStats must be collected")
	}
}

func TestCollectServiceCalls_directAndAlias(t *testing.T) {
	src := `
func (h *Handler) A() {
	h.service.DirectSvc(ctx)
	svc := h.service
	svc.AliasSvc(ctx)
}`
	got := gen.CollectServiceCalls(src)
	for _, want := range []string{"DirectSvc", "AliasSvc"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing service method %q in %v", want, got)
		}
	}
}

func TestCollectServiceCalls_skipUnexported(t *testing.T) {
	src := `
func (h *Handler) A() {
	svc := h.service
	_ = svc.unexported(ctx)
}`
	got := gen.CollectServiceCalls(src)
	if _, ok := got["unexported"]; ok {
		t.Error("unexported method must not be collected via alias path")
	}
}

func TestPortPrefix_knownDirectories(t *testing.T) {
	cases := []struct {
		dir    string
		want   string
	}{
		{"users", "Users"},
		{"api_keys", "ApiKeys"},
		{"orders", "Orders"},
	}
	for _, tc := range cases {
		got := gen.PortPrefix(tc.dir)
		if got != tc.want {
			t.Errorf("dir=%q: expected prefix %q, got %q", tc.dir, tc.want, got)
		}
	}
}

func TestDetectModulePath_readsGoMod(t *testing.T) {
	// Create a temp dir with a go.mod file.
	dir := t.TempDir()
	gomod := "module github.com/test/myapp\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	got, err := gen.DetectModulePath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github.com/test/myapp" {
		t.Errorf("expected github.com/test/myapp, got %q", got)
	}
}

func TestDetectModulePath_errorWhenNoGoMod(t *testing.T) {
	dir := t.TempDir()
	_, err := gen.DetectModulePath(dir)
	if err == nil {
		t.Error("expected error when go.mod is absent")
	}
}
