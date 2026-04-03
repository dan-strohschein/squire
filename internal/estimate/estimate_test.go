package estimate

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
	if len(result.Symbols) == 0 {
		t.Error("expected at least one matched symbol")
	}
	if result.Size == "" {
		t.Error("expected non-empty Size estimate")
	}
	if result.Size == "UNCLEAR" {
		t.Error("expected a clear estimate for a known symbol")
	}
}

func TestFromSymbols_UnknownSymbol(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"CompletelyBogusName"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if !result.Unclear {
		t.Error("expected Unclear=true for unmatched symbol")
	}
	if result.Size != "UNCLEAR" {
		t.Errorf("expected Size=UNCLEAR, got %s", result.Size)
	}
	if len(result.UnmatchedSymbols) == 0 {
		t.Error("expected at least one unmatched symbol")
	}
}

func TestFromSymbols_MultipleSymbols(t *testing.T) {
	aidDir := testutil.AidDir(t)

	result, err := FromSymbols(aidDir, []string{"Config", "NewServer"})
	if err != nil {
		t.Fatalf("FromSymbols error: %v", err)
	}
	if len(result.Symbols) < 2 {
		t.Errorf("expected at least 2 matched symbols, got %d", len(result.Symbols))
	}
}

func TestFromSymbols_InvalidDir(t *testing.T) {
	_, err := FromSymbols("/nonexistent", []string{"Config"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
