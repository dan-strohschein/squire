// Package impact analyzes the blast radius of code changes by comparing
// the semantic graph's dependency fan-out against what's actually been modified.
// It answers: "What depends on what I changed that I haven't updated yet?"
package impact

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/cartograph/pkg/query"
)

// Result holds the impact analysis output.
type Result struct {
	// What changed (from git diff or explicit input)
	ChangedFiles   []string
	ChangedSymbols []SymbolInfo

	// Full blast radius from graph analysis
	AffectedFiles     []string
	AffectedFunctions int
	AffectedPackages  []string

	// The gap — things that depend on your changes but aren't in your diff
	MissedFiles     []string
	MissedSymbols   []SymbolInfo
	MissedPackages  []string

	// Risk assessment
	Risk     string // LOW, MEDIUM, HIGH, CRITICAL
	Warnings []string
}

// SymbolInfo describes a symbol found in the analysis.
type SymbolInfo struct {
	Name       string
	Kind       string
	Module     string
	SourceFile string
	Reason     string // why it's affected: "calls ChangedFn", "depends on ChangedType", etc.
}

// FromGitDiff analyzes the impact of uncommitted or staged changes.
func FromGitDiff(projectDir, aidDir string, staged bool) (*Result, error) {
	// Get changed files from git
	changedFiles, err := getChangedFiles(projectDir, staged)
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		return &Result{Risk: "LOW", Warnings: []string{"No changes detected"}}, nil
	}

	return analyzeChangedFiles(projectDir, aidDir, changedFiles)
}

// FromFiles analyzes the impact of specific changed files.
func FromFiles(projectDir, aidDir string, files []string) (*Result, error) {
	return analyzeChangedFiles(projectDir, aidDir, files)
}

// FromSymbols analyzes the impact of changing specific symbols.
func FromSymbols(aidDir string, symbols []string) (*Result, error) {
	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 20)
	result := &Result{}

	// Resolve symbols and find their dependents
	allAffected := map[graph.NodeID]graph.Node{}
	changedSet := map[graph.NodeID]bool{}

	for _, sym := range symbols {
		nodes := g.NodesByName(sym)
		if len(nodes) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("symbol not found in graph: %s", sym))
			continue
		}

		node := nodes[0]
		result.ChangedSymbols = append(result.ChangedSymbols, SymbolInfo{
			Name:       node.Name,
			Kind:       string(node.Kind),
			Module:     node.Module,
			SourceFile: node.SourceFile,
		})
		changedSet[node.ID] = true

		// Find everything that depends on this symbol
		collectDependents(engine, node, allAffected)
	}

	// Build the result
	buildResult(result, allAffected, changedSet, nil)
	return result, nil
}

func analyzeChangedFiles(projectDir, aidDir string, changedFiles []string) (*Result, error) {
	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 20)
	result := &Result{ChangedFiles: changedFiles}

	// Map changed files to graph symbols
	changedFileSet := map[string]bool{}
	for _, f := range changedFiles {
		changedFileSet[f] = true
		// Also try relative forms
		if abs, err := filepath.Abs(filepath.Join(projectDir, f)); err == nil {
			changedFileSet[abs] = true
		}
		changedFileSet[filepath.Base(f)] = true
	}

	// Find all graph nodes whose source_file is in the changed set
	changedNodes := map[graph.NodeID]bool{}
	for _, node := range g.AllNodes() {
		if node.SourceFile == "" {
			continue
		}
		if changedFileSet[node.SourceFile] || changedFileSet[filepath.Base(node.SourceFile)] {
			changedNodes[node.ID] = true
			result.ChangedSymbols = append(result.ChangedSymbols, SymbolInfo{
				Name:       node.Name,
				Kind:       string(node.Kind),
				Module:     node.Module,
				SourceFile: node.SourceFile,
			})
		}
	}

	// For each changed symbol, find dependents
	allAffected := map[graph.NodeID]graph.Node{}
	for id := range changedNodes {
		node, err := g.NodeByID(id)
		if err != nil {
			continue
		}
		collectDependents(engine, node, allAffected)
	}

	buildResult(result, allAffected, changedNodes, changedFileSet)
	return result, nil
}

func collectDependents(engine *query.QueryEngine, node graph.Node, affected map[graph.NodeID]graph.Node) {
	switch node.Kind {
	case graph.KindType, graph.KindTrait:
		// Find everything that depends on this type
		qr, err := engine.TypeDependents(node.Name)
		if err == nil {
			for _, path := range qr.Paths {
				for _, n := range path.Nodes {
					affected[n.ID] = n
				}
			}
		}

	case graph.KindFunction, graph.KindMethod:
		// Find all callers
		qr, err := engine.CallStack(node.Name, query.Reverse)
		if err == nil {
			for _, path := range qr.Paths {
				for _, n := range path.Nodes {
					affected[n.ID] = n
				}
			}
		}
	}

	// Always include the node itself
	affected[node.ID] = node
}

func buildResult(result *Result, allAffected map[graph.NodeID]graph.Node, changedNodes map[graph.NodeID]bool, changedFiles map[string]bool) {
	affectedFiles := map[string]bool{}
	affectedPkgs := map[string]bool{}
	missedFiles := map[string]bool{}
	missedPkgs := map[string]bool{}

	for id, node := range allAffected {
		if node.SourceFile != "" {
			affectedFiles[node.SourceFile] = true
		}
		affectedPkgs[node.Module] = true

		if node.Kind == graph.KindFunction || node.Kind == graph.KindMethod {
			result.AffectedFunctions++
		}

		// Is this node outside the changed set?
		if !changedNodes[id] {
			inChangedFile := false
			if changedFiles != nil && node.SourceFile != "" {
				inChangedFile = changedFiles[node.SourceFile] || changedFiles[filepath.Base(node.SourceFile)]
			}

			if !inChangedFile {
				if node.Kind == graph.KindFunction || node.Kind == graph.KindMethod || node.Kind == graph.KindType || node.Kind == graph.KindTrait {
					reason := "depends on changed code"
					result.MissedSymbols = append(result.MissedSymbols, SymbolInfo{
						Name:       node.Name,
						Kind:       string(node.Kind),
						Module:     node.Module,
						SourceFile: node.SourceFile,
						Reason:     reason,
					})
				}
				if node.SourceFile != "" {
					missedFiles[node.SourceFile] = true
				}
				missedPkgs[node.Module] = true
			}
		}
	}

	for f := range affectedFiles {
		result.AffectedFiles = append(result.AffectedFiles, f)
	}
	for p := range affectedPkgs {
		result.AffectedPackages = append(result.AffectedPackages, p)
	}
	for f := range missedFiles {
		result.MissedFiles = append(result.MissedFiles, f)
	}
	for p := range missedPkgs {
		result.MissedPackages = append(result.MissedPackages, p)
	}

	// Compute risk
	result.Risk = computeRisk(result)
}

func computeRisk(r *Result) string {
	missed := len(r.MissedSymbols)

	if missed == 0 {
		return "LOW"
	}
	if missed <= 5 && len(r.MissedPackages) <= 1 {
		return "MEDIUM"
	}
	if missed <= 15 && len(r.MissedPackages) <= 3 {
		return "HIGH"
	}
	return "CRITICAL"
}

// --- Git helpers ---

func getChangedFiles(dir string, staged bool) ([]string, error) {
	var args []string
	if staged {
		args = []string{"diff", "--cached", "--name-only"}
	} else {
		args = []string{"diff", "--name-only", "HEAD"}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Fallback: try unstaged changes
		cmd2 := exec.Command("git", "diff", "--name-only")
		cmd2.Dir = dir
		out2, err2 := cmd2.Output()
		if err2 != nil {
			// Try status for untracked
			cmd3 := exec.Command("git", "status", "--porcelain")
			cmd3.Dir = dir
			out3, err3 := cmd3.Output()
			if err3 != nil {
				return nil, err3
			}
			return parseStatusOutput(string(out3)), nil
		}
		out = out2
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasSuffix(line, ".go") {
			files = append(files, line)
		}
	}
	return files, nil
}

func parseStatusOutput(output string) []string {
	var files []string
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 4 {
			continue
		}
		file := strings.TrimSpace(line[3:])
		if strings.HasSuffix(file, ".go") {
			files = append(files, file)
		}
	}
	return files
}

// FindAidocs walks up from dir looking for .aidocs/.
func FindAidocs(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for d := abs; ; d = filepath.Dir(d) {
		candidate := filepath.Join(d, ".aidocs")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
	}
	return ""
}
