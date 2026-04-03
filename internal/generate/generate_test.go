package generate

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestLoadGraphStats_CachedLoading(t *testing.T) {
	aidDir := testutil.AidDir(t)

	stats, err := LoadGraphStats(aidDir)
	if err != nil {
		t.Fatalf("LoadGraphStats error: %v", err)
	}
	if stats == nil {
		t.Fatal("LoadGraphStats returned nil")
	}
	if stats.NodeCount == 0 {
		t.Error("expected non-zero NodeCount")
	}
	if stats.EdgeCount == 0 {
		t.Error("expected non-zero EdgeCount")
	}
	if stats.CallEdges == 0 {
		t.Error("expected non-zero CallEdges (fixture has @calls)")
	}

	// Call again — should use cache and still succeed
	stats2, err := LoadGraphStats(aidDir)
	if err != nil {
		t.Fatalf("LoadGraphStats (second call) error: %v", err)
	}
	if stats2.NodeCount != stats.NodeCount {
		t.Errorf("cached load returned different NodeCount: %d vs %d", stats2.NodeCount, stats.NodeCount)
	}
}

func TestLoadGraphStats_InvalidDir(t *testing.T) {
	_, err := LoadGraphStats("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestFindGenerator_Unknown(t *testing.T) {
	_, err := FindGenerator("cobol")
	if err == nil {
		t.Fatal("expected error for unknown generator")
	}
}
