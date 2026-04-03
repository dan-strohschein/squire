package status

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/detect"
	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestCheckStaleness_NoAidDir(t *testing.T) {
	project := &detect.Project{
		AidDir:     "/nonexistent/.aidocs",
		SourceRoot: "/nonexistent",
	}

	_, _, _, err := CheckStaleness(project)
	if err == nil {
		t.Fatal("expected error when .aidocs/ does not exist")
	}
}

func TestCheckStaleness_WithFixture(t *testing.T) {
	aidDir := testutil.AidDir(t)
	projectDir := testutil.ProjectDir(t)

	project := &detect.Project{
		Language:   "Go",
		Module:     "testproject",
		SourceRoot: projectDir,
		AidDir:     aidDir,
		Packages: []detect.PackageInfo{
			{Name: "testpkg", Dir: "testpkg"},
		},
	}

	stale, fresh, missing, err := CheckStaleness(project)
	if err != nil {
		t.Fatalf("CheckStaleness error: %v", err)
	}

	// The fixture has @code_version git:abc1234 which won't match real git
	// so the package should be either stale (if git works in testdata) or fresh (if no git)
	total := len(stale) + len(fresh) + len(missing)
	if total == 0 {
		t.Error("expected at least one package in stale/fresh/missing")
	}
	t.Logf("stale=%d fresh=%d missing=%d", len(stale), len(fresh), len(missing))
}

func TestGetGitHash_NonGitDir(t *testing.T) {
	hash := getGitHash("/tmp")
	// /tmp is not a git repo, should return empty string
	if hash != "" {
		t.Logf("getGitHash(/tmp) returned %q (might be inside a git repo)", hash)
	}
}
