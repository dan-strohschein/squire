// Package testutil provides shared helpers for squire tests.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// AidDir returns the absolute path to the shared testdata/aidocs/ fixture directory.
func AidDir(t *testing.T) string {
	t.Helper()
	// Walk up from this file to find the project root
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is internal/testutil/testutil.go → project root is ../../
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	aidDir := filepath.Join(root, "testdata", "aidocs")
	if _, err := os.Stat(aidDir); err != nil {
		t.Fatalf("testdata/aidocs/ not found at %s: %v", aidDir, err)
	}
	abs, _ := filepath.Abs(aidDir)
	return abs
}

// ProjectDir returns the absolute path to the testdata/ directory (parent of aidocs/).
func ProjectDir(t *testing.T) string {
	t.Helper()
	return filepath.Dir(AidDir(t))
}
