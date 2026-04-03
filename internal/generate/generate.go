package generate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
	"github.com/dan-strohschein/aidkit/pkg/validator"
	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/squire/internal/detect"
)

// Result holds the outcome of a generation run.
type Result struct {
	PackagesProcessed int
	FilesGenerated    int
	Warnings          []string
}

// GraphStats holds summary stats from the loaded graph.
type GraphStats struct {
	NodeCount int
	EdgeCount int
	CallEdges int
}

// Generate runs L1 AID generation for the given project.
func Generate(project *detect.Project) (*Result, error) {
	if err := os.MkdirAll(project.AidDir, 0755); err != nil {
		return nil, fmt.Errorf("create .aidocs/: %w", err)
	}

	switch project.Language {
	case "Go":
		return generateGo(project)
	case "TypeScript":
		return generateExternal(project, "aid-gen-ts")
	case "Python":
		return generateExternal(project, "aid-gen")
	case "C#":
		return generateExternal(project, "aid-gen-cs")
	default:
		return nil, fmt.Errorf("unsupported language: %s", project.Language)
	}
}

func generateGo(project *detect.Project) (*Result, error) {
	genPath, err := FindGenerator("go")
	if err != nil {
		return nil, fmt.Errorf("aid-gen-go not found: %w\nInstall it with: go install github.com/dan-strohschein/aid-gen-go@latest", err)
	}

	result := &Result{}

	// Detect duplicate package names and build unique file names.
	// e.g., audit/handlers → audit_handlers.aid, export/handlers → export_handlers.aid
	nameCount := map[string]int{}
	for _, pkg := range project.Packages {
		nameCount[pkg.Name]++
	}
	aidFileName := func(pkg detect.PackageInfo) string {
		if nameCount[pkg.Name] > 1 {
			// Use directory path with underscores: internal/audit/handlers → audit_handlers
			clean := strings.ReplaceAll(pkg.Dir, string(filepath.Separator), "_")
			clean = strings.TrimPrefix(clean, "internal_")
			clean = strings.TrimPrefix(clean, "src_internal_")
			clean = strings.TrimPrefix(clean, "src_")
			clean = strings.TrimPrefix(clean, "pkg_")
			return clean
		}
		return pkg.Name
	}

	// Generate each package into a temp directory, then move to final name.
	// This prevents packages with the same Go name from overwriting each other.
	tmpDir, err := os.MkdirTemp("", "squire-gen-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, pkg := range project.Packages {
		pkgDir := filepath.Join(project.SourceRoot, pkg.Dir)

		// Generate into temp dir (aid-gen-go writes <pkgname>.aid)
		cmd := exec.Command(genPath, "--output", tmpDir, pkgDir)
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to generate %s: %v", pkg.Name, err))
			continue
		}

		// Move from temp to final location with unique name
		generatedPath := filepath.Join(tmpDir, pkg.Name+".aid")
		uniqueName := aidFileName(pkg)
		targetPath := filepath.Join(project.AidDir, uniqueName+".aid")

		if _, err := os.Stat(generatedPath); err == nil {
			data, readErr := os.ReadFile(generatedPath)
			if readErr == nil {
				os.WriteFile(targetPath, data, 0644)
			}
			os.Remove(generatedPath) // clean up temp
		}

		stampCodeVersion(targetPath, pkgDir)

		result.PackagesProcessed++
		result.FilesGenerated++
	}

	// Generate manifest
	if err := generateManifest(project); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("manifest generation failed: %v", err))
	}

	// Validate all generated files using aidkit's rule-based validator
	aids, _ := filepath.Glob(filepath.Join(project.AidDir, "*.aid"))
	for _, aidPath := range aids {
		af, parseWarns, err := parser.ParseFile(aidPath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("parse error in %s: %v", filepath.Base(aidPath), err))
			continue
		}
		for _, w := range parseWarns {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", filepath.Base(aidPath), w.String()))
		}
		// Run spec-level validation rules
		issues := validator.Validate(af)
		for _, issue := range issues {
			if issue.Severity >= validator.SeverityWarning {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", filepath.Base(aidPath), issue.String()))
			}
		}
	}

	return result, nil
}

func generateExternal(project *detect.Project, generatorName string) (*Result, error) {
	genPath, err := FindGenerator(generatorName)
	if err != nil {
		return nil, fmt.Errorf("%s not found: %w\nInstall it with: squire install %s", generatorName, err, generatorName)
	}

	cmd := exec.Command(genPath, "--output", project.AidDir, project.SourceRoot)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("generator failed: %w", err)
	}

	aids, _ := filepath.Glob(filepath.Join(project.AidDir, "*.aid"))
	return &Result{
		PackagesProcessed: len(aids),
		FilesGenerated:    len(aids),
	}, nil
}

// FindGenerator looks for a generator binary on PATH or in common locations.
func FindGenerator(language string) (string, error) {
	var names []string
	switch language {
	case "go":
		names = []string{"aid-gen-go"}
	case "typescript", "ts":
		names = []string{"aid-gen-ts"}
	case "python", "py":
		names = []string{"aid-gen"}
	case "csharp", "cs":
		names = []string{"aid-gen-cs"}
	default:
		names = []string{language}
	}

	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
		for _, dir := range []string{"/usr/local/bin", filepath.Join(os.Getenv("HOME"), "go", "bin")} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("generator %q not found on PATH", names[0])
}

func generateManifest(project *detect.Project) error {
	manifestPath := filepath.Join(project.AidDir, "manifest.aid")

	var b strings.Builder
	b.WriteString("@manifest\n")
	b.WriteString(fmt.Sprintf("@project %s\n", filepath.Base(project.Module)))
	b.WriteString("@aid_version 0.1\n")
	b.WriteString(fmt.Sprintf("@lang %s\n", strings.ToLower(project.Language)))
	b.WriteString("\n")

	aids, _ := filepath.Glob(filepath.Join(project.AidDir, "*.aid"))
	for _, aidPath := range aids {
		base := filepath.Base(aidPath)
		if base == "manifest.aid" {
			continue
		}
		name := strings.TrimSuffix(base, ".aid")
		b.WriteString("---\n\n")
		b.WriteString(fmt.Sprintf("@package %s\n", name))
		b.WriteString(fmt.Sprintf("@aid_file %s\n", base))
		b.WriteString("@layer skeleton\n")
		b.WriteString("\n")
	}

	return os.WriteFile(manifestPath, []byte(b.String()), 0644)
}

// stampCodeVersion injects @code_version git:<hash> into an .aid file header.
func stampCodeVersion(aidPath, sourceDir string) {
	hash := getGitHash(sourceDir)
	if hash == "" {
		return
	}

	content, err := os.ReadFile(aidPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	stamped := false

	for _, line := range lines {
		// Replace existing @code_version or inject after @aid_version
		if strings.HasPrefix(line, "@code_version ") {
			result = append(result, "@code_version git:"+hash)
			stamped = true
		} else {
			result = append(result, line)
			if !stamped && strings.HasPrefix(line, "@aid_version ") {
				result = append(result, "@code_version git:"+hash)
				stamped = true
			}
		}
	}

	// If no @aid_version found, prepend after first blank line or separator
	if !stamped {
		for i, line := range result {
			if line == "" || line == "---" {
				result = append(result[:i+1], append([]string{"@code_version git:" + hash}, result[i+1:]...)...)
				break
			}
		}
	}

	os.WriteFile(aidPath, []byte(strings.Join(result, "\n")), 0644)
}

func getGitHash(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%h", "--", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LoadGraphStats loads .aidocs/ into the embedded cartograph graph engine.
func LoadGraphStats(aidDir string) (*GraphStats, error) {
	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, err
	}

	stats := g.Stats()
	callEdges := 0
	if v, ok := stats.EdgesByKind[string(graph.EdgeCalls)]; ok {
		callEdges = v
	}

	return &GraphStats{
		NodeCount: stats.NodeCount,
		EdgeCount: stats.EdgeCount,
		CallEdges: callEdges,
	}, nil
}
