package gen

import "testing"

func TestLoadCallerScanContext_disabledDoesNotLoadModule(t *testing.T) {
	ctx, err := loadCallerScanContext(false)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.enabled {
		t.Fatal("disabled caller scan context should not be enabled")
	}
	if ctx.modulePath != "" {
		t.Fatalf("modulePath = %q, want empty", ctx.modulePath)
	}
	if len(ctx.modulePkgs) != 0 {
		t.Fatalf("modulePkgs len = %d, want 0", len(ctx.modulePkgs))
	}
}
