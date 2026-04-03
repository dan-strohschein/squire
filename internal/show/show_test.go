package show

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestSymbol_Found(t *testing.T) {
	aidDir := testutil.AidDir(t)
	projectDir := testutil.ProjectDir(t)

	results, err := Symbol(aidDir, projectDir, "Server.Start")
	if err != nil {
		t.Fatalf("Symbol(Server.Start) error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for Server.Start")
	}

	r := results[0]
	if r.Name != "Server.Start" {
		t.Errorf("expected Name=Server.Start, got %s", r.Name)
	}
	if r.Module != "testpkg" {
		t.Errorf("expected Module=testpkg, got %s", r.Module)
	}
	if r.Purpose == "" {
		t.Error("expected non-empty Purpose")
	}
}

func TestSymbol_MethodByBaseName(t *testing.T) {
	aidDir := testutil.AidDir(t)
	projectDir := testutil.ProjectDir(t)

	// Searching "Start" should find "Server.Start" via fuzzy matching
	results, err := Symbol(aidDir, projectDir, "Start")
	if err != nil {
		t.Fatalf("Symbol(Start) error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected fuzzy match for 'Start' to find Server.Start")
	}

	found := false
	for _, r := range results {
		if r.Name == "Server.Start" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Server.Start in fuzzy results for 'Start'")
	}
}

func TestSymbol_NotFound(t *testing.T) {
	aidDir := testutil.AidDir(t)
	projectDir := testutil.ProjectDir(t)

	_, err := Symbol(aidDir, projectDir, "NonexistentSymbol")
	if err == nil {
		t.Fatal("expected error for nonexistent symbol")
	}
}

func TestSymbol_Type(t *testing.T) {
	aidDir := testutil.AidDir(t)
	projectDir := testutil.ProjectDir(t)

	results, err := Symbol(aidDir, projectDir, "Config")
	if err != nil {
		t.Fatalf("Symbol(Config) error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for Config")
	}
	if results[0].Kind != "Type" {
		t.Errorf("expected Kind=Type, got %s", results[0].Kind)
	}
}

func TestResolveSourcePath_NotFound(t *testing.T) {
	result := ResolveSourcePath("nonexistent/file.go", "/tmp")
	if result != "" {
		t.Errorf("expected empty string for missing file, got %s", result)
	}
}

func TestExtractBlock_InvalidFile(t *testing.T) {
	_, _, err := ExtractBlock("/nonexistent/file.go", 1)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
