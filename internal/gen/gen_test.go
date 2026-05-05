package gen_test

// gen_test.go — контрактные тесты для internal/gen.
//
// Контракт генератора:
//  1. CollectRepoCalls находит все методы репозитория, вызываемые в коде.
//  2. CollectServiceCalls находит все методы сервиса, вызываемые в коде.
//  3. MethodSigs парсит AST файла и возвращает сигнатуры методов.
//  4. PortPrefix строит правильный префикс из имени директории.
//  5. InferPortPrefix подхватывает WebPush из полей *Service/*Handler при «webpush» в имени папки.
//  6. DetectModulePath читает module из go.mod в заданном корне.
//  7. LoadSources читает не-тестовые .go файлы пакета, исключая ports.go и адаптеры.

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCollectRepoCallsWithAllowed_directAndAlias(t *testing.T) {
	src := `
func (s *Service) DoWork(ctx context.Context) error {
	s.storage.DirectCall(ctx)
	store := s.storage
	store.AliasCall(ctx)
	return nil
}`
	got := gen.CollectRepoCallsWithAllowed(src, []string{"*.storage.*"})
	for _, want := range []string{"DirectCall", "AliasCall"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing repo method %q in %v", want, got)
		}
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

func TestCollectServiceCallsWithAllowed_directAndAlias(t *testing.T) {
	src := `
func (h *Handler) A() {
	h.svc.DirectSvc(ctx)
	service := h.svc
	service.AliasSvc(ctx)
}`
	got := gen.CollectServiceCallsWithAllowed(src, []string{"*.svc.*"})
	for _, want := range []string{"DirectSvc", "AliasSvc"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing service method %q in %v", want, got)
		}
	}
}

func TestCollectServiceCallsWithAllowed_exactMethod(t *testing.T) {
	src := `
func (h *Handler) A() {
	h.svc.DirectSvc(ctx)
	h.svc.Other(ctx)
}`
	got := gen.CollectServiceCallsWithAllowed(src, []string{"*.svc.DirectSvc"})
	if _, ok := got["DirectSvc"]; !ok {
		t.Errorf("missing exact service method in %v", got)
	}
	if _, ok := got["Other"]; ok {
		t.Errorf("unexpected non-matching service method in %v", got)
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
		dir  string
		want string
	}{
		{"users", "Users"},
		{"api_keys", "ApiKeys"},
		{"orders", "Orders"},
		{"webpush", "Webpush"},
	}
	for _, tc := range cases {
		got := gen.PortPrefix(tc.dir)
		if got != tc.want {
			t.Errorf("dir=%q: expected prefix %q, got %q", tc.dir, tc.want, got)
		}
	}
}

func TestInferPortPrefix_webpushFromStructFields(t *testing.T) {
	dir := t.TempDir()
	// Simulates package webpush with idiomatic WebPush* interface names.
	content := `package webpush

type WebPushRepository interface{}

type WebPushService interface{}

type Repository struct{}

type Service struct {
	repo WebPushRepository
}

type Handler struct {
	service WebPushService
}
`
	if err := os.WriteFile(filepath.Join(dir, "bundle.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, err := gen.ParsePackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := gen.InferPortPrefix(pkg)
	if !ok || got != "WebPush" {
		t.Fatalf("InferPortPrefix: got %q, ok=%v", got, ok)
	}
}

func TestInferPortPrefix_repoServiceMismatch(t *testing.T) {
	dir := t.TempDir()
	content := `package x

type FooRepository interface{}
type BarService interface{}
type Service struct { repo FooRepository }
type Handler struct { service BarService }
type Repository struct{}
`
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg, err := gen.ParsePackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := gen.InferPortPrefix(pkg); ok {
		t.Fatal("expected no single prefix when repository and service prefixes differ")
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

func TestLoadSources_includesNonTestGoFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"service.go":    "package x\nfunc f() {}\n",
		"handler.go":    "package x\nfunc g() {}\n",
		"repository.go": "package x\nfunc h() {}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	bodies, err := gen.LoadSources(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodies) != 3 {
		t.Fatalf("expected 3 source bodies, got %d: %v", len(bodies), bodies)
	}
}

func TestLoadSources_skipsExcludedFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"service.go":          "package x\n",
		"service_test.go":     "package x_test\n",
		"ports.go":            "package x\n",
		"adapters.go":         "package x\n",
		"foo_adapter.go":      "package x\n",
		"bar_adapter_test.go": "package x_test\n",
		"README.md":           "skip me",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	bodies, err := gen.LoadSources(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodies) != 1 {
		t.Fatalf("expected only service.go, got %d bodies: %v", len(bodies), bodies)
	}
	if !strings.Contains(bodies[0], "package x") {
		t.Errorf("expected service.go body, got %q", bodies[0])
	}
}

func TestLoadSources_emptyDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	bodies, err := gen.LoadSources(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodies) != 0 {
		t.Errorf("expected no bodies for empty dir, got %d", len(bodies))
	}
}

func TestLoadSources_errorWhenDirMissing(t *testing.T) {
	_, err := gen.LoadSources("/nonexistent/portsmith/test/dir")
	if err == nil {
		t.Error("expected error when directory does not exist")
	}
}

// --- Cross-module caller scanning ---

// makeFixtureModule writes a fake module tree with go.mod and the supplied
// files (relative paths → contents) under root. Files must be valid Go since
// LoadModulePackages performs full type-checking.
func makeFixtureModule(t *testing.T, root, modulePath string, files map[string]string) {
	t.Helper()
	gomod := "module " + modulePath + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for rel, body := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// botsFixture is a minimal target package used by CollectCrossModuleCalls
// tests. Methods listed here exist on *Service / *Repository so type-checking
// of caller code succeeds.
const botsFixture = `package bots

type Service struct{}

func (s *Service) GetSubscriber()      {}
func (s *Service) SetTokenValidator()  {}
func (s *Service) CleanupOldMessages() {}
func (s *Service) AliasField()         {}

type Repository struct{}

func (r *Repository) LoadAll()        {}
func (r *Repository) AliasRepoField() {}

func NewService() *Service       { return &Service{} }
func NewRepository() *Repository { return &Repository{} }
`

func TestCollectCrossModuleCalls_structFieldPattern(t *testing.T) {
	root := t.TempDir()
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/chat/service.go": `package chat

import "example.com/mod/internal/bots"

type Service struct {
	bots *bots.Service
	repo *bots.Repository
}

func (s *Service) Handle() {
	s.bots.GetSubscriber()
	s.repo.LoadAll()
}
`,
	})

	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	repo, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	if _, ok := svc["GetSubscriber"]; !ok {
		t.Errorf("expected GetSubscriber, got svc=%v", svc)
	}
	if _, ok := repo["LoadAll"]; !ok {
		t.Errorf("expected LoadAll, got repo=%v", repo)
	}
}

func TestCollectCrossModuleCalls_directIdentPattern(t *testing.T) {
	root := t.TempDir()
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/app/wire.go": `package app

import "example.com/mod/internal/bots"

func wire(b *bots.Service) {
	b.SetTokenValidator()
}

func startup() {
	svc := bots.NewService()
	svc.CleanupOldMessages()
}
`,
	})

	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	_, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	for _, want := range []string{"SetTokenValidator", "CleanupOldMessages"} {
		if _, ok := svc[want]; !ok {
			t.Errorf("expected %s in svc, got %v", want, svc)
		}
	}
}

func TestCollectCrossModuleCalls_noTargetImport_noResults(t *testing.T) {
	root := t.TempDir()
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/a/service.go": `package a

import "fmt"

func A() { fmt.Println() }
`,
	})

	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	repo, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	if len(repo) != 0 || len(svc) != 0 {
		t.Errorf("expected empty results, got repo=%v svc=%v", repo, svc)
	}
}

func TestCollectCrossModuleCalls_typeAliasInSiblingFile(t *testing.T) {
	root := t.TempDir()
	// external_ports.go re-exports bots.Service as a local alias to dodge
	// cross-imports linting in service.go (cdp-backend pattern). go/types
	// resolves aliases precisely.
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/inbox/external_ports.go": `package inbox

import "example.com/mod/internal/bots"

type (
	BotsService    = bots.Service
	BotsRepository = bots.Repository
)
`,
		"internal/inbox/service.go": `package inbox

type Service struct {
	bots *BotsService
	repo *BotsRepository
}

func (s *Service) Run() {
	s.bots.AliasField()
	s.repo.AliasRepoField()
}
`,
	})

	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	repo, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	if _, ok := svc["AliasField"]; !ok {
		t.Errorf("expected AliasField via type alias, got svc=%v", svc)
	}
	if _, ok := repo["AliasRepoField"]; !ok {
		t.Errorf("expected AliasRepoField via type alias, got repo=%v", repo)
	}
}

// Regression guard: fields with identical names but different types on
// different structs in the same file must not be conflated. The previous
// regex/AST-heuristic approach failed on this; go/types resolves field
// types exactly per receiver.
func TestCollectCrossModuleCalls_disambiguatesSameNamedFields(t *testing.T) {
	root := t.TempDir()
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/integrations/integrations.go": `package integrations

type Service struct{}

func (s *Service) ListForProject() {}
`,
		"internal/assistant/adapters.go": `package assistant

import (
	"example.com/mod/internal/bots"
	"example.com/mod/internal/integrations"
)

type IntegrationsAdapter struct{ service *integrations.Service }

func (a *IntegrationsAdapter) Foo() { a.service.ListForProject() }

type BotsAdapter struct{ service *bots.Service }

func (a *BotsAdapter) Bar() { a.service.GetSubscriber() }
`,
	})

	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	_, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	if _, ok := svc["GetSubscriber"]; !ok {
		t.Errorf("expected GetSubscriber, got %v", svc)
	}
	if _, ok := svc["ListForProject"]; ok {
		t.Errorf("ListForProject must NOT leak into bots methods (it belongs to integrations.Service)")
	}
}

func TestCollectCrossModuleCalls_skipsTargetPackageItself(t *testing.T) {
	root := t.TempDir()
	makeFixtureModule(t, root, "example.com/mod", map[string]string{
		"internal/bots/bots.go": botsFixture,
		"internal/bots/intra.go": `package bots

func intra() {
	s := &Service{}
	s.GetSubscriber()
}
`,
	})
	pkgs, err := gen.LoadModulePackages(root)
	if err != nil {
		t.Fatalf("LoadModulePackages: %v", err)
	}
	_, svc := gen.CollectCrossModuleCalls(pkgs, "example.com/mod/internal/bots")
	if _, ok := svc["GetSubscriber"]; ok {
		t.Errorf("intra-module call must not be reported as cross-module: %v", svc)
	}
}
