package status

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
	"github.com/dan-strohschein/squire/internal/detect"
)

// CheckStaleness compares .aid files against current git state.
// Returns lists of stale, fresh, and missing package names.
func CheckStaleness(project *detect.Project) (stale, fresh, missing []string, err error) {
	if _, err := os.Stat(project.AidDir); err != nil {
		return nil, nil, nil, err
	}

	// Build a set of existing .aid files (by basename without extension)
	aidFiles := map[string]string{} // name → code_version
	aids, _ := filepath.Glob(filepath.Join(project.AidDir, "*.aid"))
	for _, aidPath := range aids {
		base := strings.TrimSuffix(filepath.Base(aidPath), ".aid")
		if base == "manifest" {
			continue
		}

		codeVersion := ""
		af, _, parseErr := parser.ParseFile(aidPath)
		if parseErr == nil && af.Header.CodeVersion != "" {
			codeVersion = strings.TrimPrefix(af.Header.CodeVersion, "git:")
		}
		aidFiles[base] = codeVersion
	}

	// For each detected package, check if an .aid file exists
	for _, pkg := range project.Packages {
		_, hasAid := aidFiles[pkg.Name]

		if !hasAid {
			missing = append(missing, pkg.Name)
			continue
		}

		codeVersion := aidFiles[pkg.Name]
		if codeVersion == "" {
			// .aid file exists but no code_version — treat as "generated, not versioned"
			fresh = append(fresh, pkg.Name)
			continue
		}

		// Compare against current git hash for this package's directory
		pkgDir := filepath.Join(project.SourceRoot, pkg.Dir)
		gitHash := getGitHash(pkgDir)

		if gitHash == "" || gitHash == codeVersion {
			fresh = append(fresh, pkg.Name)
		} else {
			stale = append(stale, pkg.Name)
		}

		// Remove from aidFiles so we can detect orphans later
		delete(aidFiles, pkg.Name)
	}

	return stale, fresh, missing, nil
}

// getGitHash returns the short git hash for the latest change in a directory.
func getGitHash(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%h", "--", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
