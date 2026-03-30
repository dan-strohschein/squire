package detect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project holds detected project metadata.
type Project struct {
	Language     string // "Go", "TypeScript", "Python", "C#"
	Evidence     string // what file triggered detection, e.g. "found go.mod"
	Module       string // module/package name
	SourceRoot   string // root directory of source
	AidDir       string // .aidocs/ path
	PackageCount int    // number of detected packages
	Packages     []PackageInfo
}

// PackageInfo describes a single package within the project.
type PackageInfo struct {
	Name     string
	Dir      string // relative path from project root
	AidName  string // unique name for the .aid file (set by generate)
}

// Detect analyzes a directory and returns project metadata.
func Detect(dir string) (*Project, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Try Go
	if p, err := detectGo(absDir); err == nil {
		return p, nil
	}

	// Try TypeScript
	if p, err := detectTypeScript(absDir); err == nil {
		return p, nil
	}

	// Try Python
	if p, err := detectPython(absDir); err == nil {
		return p, nil
	}

	// Try C#
	if p, err := detectCSharp(absDir); err == nil {
		return p, nil
	}

	return nil, fmt.Errorf("could not detect project language in %s (no go.mod, package.json, pyproject.toml, or .csproj found)", dir)
}

func detectGo(dir string) (*Project, error) {
	gomod := filepath.Join(dir, "go.mod")
	f, err := os.Open(gomod)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Extract module name from first line
	scanner := bufio.NewScanner(f)
	var module string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			module = strings.TrimPrefix(line, "module ")
			break
		}
	}
	if module == "" {
		module = filepath.Base(dir)
	}

	// Find all Go packages
	packages := findGoPackages(dir)

	return &Project{
		Language:     "Go",
		Evidence:     "found go.mod",
		Module:       module,
		SourceRoot:   dir,
		AidDir:       filepath.Join(dir, ".aidocs"),
		PackageCount: len(packages),
		Packages:     packages,
	}, nil
}

func findGoPackages(root string) []PackageInfo {
	var packages []PackageInfo
	seen := map[string]bool{}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden dirs, vendor, testdata
		name := info.Name()
		if info.IsDir() {
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for .go files (not test files)
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		dir := filepath.Dir(path)
		if seen[dir] {
			return nil
		}
		seen[dir] = true

		rel, _ := filepath.Rel(root, dir)
		if rel == "" {
			rel = "."
		}

		pkgName := filepath.Base(dir)
		packages = append(packages, PackageInfo{
			Name: pkgName,
			Dir:  rel,
		})

		return nil
	})

	return packages
}

func detectTypeScript(dir string) (*Project, error) {
	pkg := filepath.Join(dir, "package.json")
	tsconfig := filepath.Join(dir, "tsconfig.json")

	if _, err := os.Stat(pkg); err != nil {
		return nil, err
	}
	if _, err := os.Stat(tsconfig); err != nil {
		return nil, err
	}

	return &Project{
		Language:   "TypeScript",
		Evidence:   "found package.json + tsconfig.json",
		Module:     filepath.Base(dir),
		SourceRoot: dir,
		AidDir:     filepath.Join(dir, ".aidocs"),
	}, nil
}

func detectPython(dir string) (*Project, error) {
	for _, marker := range []string{"pyproject.toml", "setup.py", "setup.cfg"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return &Project{
				Language:   "Python",
				Evidence:   "found " + marker,
				Module:     filepath.Base(dir),
				SourceRoot: dir,
				AidDir:     filepath.Join(dir, ".aidocs"),
			}, nil
		}
	}
	return nil, fmt.Errorf("no Python project markers found")
}

func detectCSharp(dir string) (*Project, error) {
	// Look for .csproj or .sln
	var found string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".csproj") || strings.HasSuffix(info.Name(), ".sln") {
			found = info.Name()
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return nil, fmt.Errorf("no C# project markers found")
	}

	return &Project{
		Language:   "C#",
		Evidence:   "found " + found,
		Module:     filepath.Base(dir),
		SourceRoot: dir,
		AidDir:     filepath.Join(dir, ".aidocs"),
	}, nil
}

// FindAidocs walks up from dir looking for .aidocs/.
func FindAidocs(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for d := abs; ; d = filepath.Dir(d) {
		candidate := filepath.Join(d, ".aidocs")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
	}
	return ""
}
