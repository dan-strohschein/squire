package impact

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestFromSymbols_KnownSymbol(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"Server.Start"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if result == nil {
		t.Fatal("FromSymbols returned nil")
	}
	if len(result.ChangedSymbols) == 0 {
		t.Error("expected at least one ChangedSymbol")
	}
	if result.AffectedFunctions == 0 {
		t.Error("expected non-zero AffectedFunctions (Server.Start has callers/callees)")
	}
	if result.Risk == "" {
		t.Error("expected non-empty Risk assessment")
	}
}

func TestFromSymbols_UnknownSymbol(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"NonexistentThing"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if result == nil {
		t.Fatal("FromSymbols returned nil")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unknown symbol")
	}
}

func TestFromSymbols_TypeSymbol(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"Config"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if len(result.ChangedSymbols) == 0 {
		t.Error("expected Config to be found as a changed symbol")
	}
}

func TestFromSymbols_MultipleSymbols(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"Config", "Server.Start"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if len(result.ChangedSymbols) < 2 {
		t.Errorf("expected at least 2 ChangedSymbols, got %d", len(result.ChangedSymbols))
	}
}

func TestFromSymbols_InvalidDir(t *testing.T) {
	_, err := FromSymbols("/nonexistent", []string{"Config"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
