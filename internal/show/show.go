// Package show extracts individual function/type source code using
// AID graph metadata, avoiding the need to read entire files.
package show

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
)

// Result holds the extracted source code for a symbol.
type Result struct {
	Name       string
	Kind       string
	Module     string
	SourceFile string // relative path
	StartLine  int
	EndLine    int
	Source     string // the extracted source code
	Purpose    string // from AID @purpose
	Signature  string // from AID @sig
}

// Symbol looks up a symbol in the graph and extracts its source code.
func Symbol(aidDir, projectDir string, symbolName string) ([]Result, error) {
	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	nodes := g.NodesByName(symbolName)
	if len(nodes) == 0 {
		// Try fuzzy — look for methods matching the bare name
		for _, n := range g.AllNodes() {
			if n.Kind == graph.KindField || n.Kind == graph.KindModule || n.Kind == graph.KindConstant {
				continue
			}
			// Match "Type.Method" when searching for "Method"
			if strings.HasSuffix(n.Name, "."+symbolName) {
				nodes = append(nodes, n)
			}
		}
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("symbol not found: %s", symbolName)
	}

	var results []Result
	for _, node := range nodes {
		if node.Kind == graph.KindField || node.Kind == graph.KindModule {
			continue
		}
		if node.SourceFile == "" || node.SourceLine == 0 {
			// No source location — include AID info only
			results = append(results, Result{
				Name:      node.Name,
				Kind:      string(node.Kind),
				Module:    node.Module,
				Purpose:   node.Purpose,
				Signature: node.Signature,
			})
			continue
		}

		// Resolve the source file path
		srcPath := ResolveSourcePath(node.SourceFile, projectDir)
		if srcPath == "" {
			results = append(results, Result{
				Name:       node.Name,
				Kind:       string(node.Kind),
				Module:     node.Module,
				SourceFile: node.SourceFile,
				StartLine:  node.SourceLine,
				Purpose:    node.Purpose,
				Signature:  node.Signature,
			})
			continue
		}

		// Extract the function/type body from source
		source, endLine, err := ExtractBlock(srcPath, node.SourceLine)
		if err != nil {
			continue
		}

		results = append(results, Result{
			Name:       node.Name,
			Kind:       string(node.Kind),
			Module:     node.Module,
			SourceFile: node.SourceFile,
			StartLine:  node.SourceLine,
			EndLine:    endLine,
			Source:     source,
			Purpose:    node.Purpose,
			Signature:  node.Signature,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("symbol %s found in graph but no source available", symbolName)
	}

	return results, nil
}

// ExtractBlock reads a function or type definition starting at startLine.
// It uses brace counting to find where the block ends.
func ExtractBlock(filePath string, startLine int) (string, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	lineNum := 0
	capturing := false
	braceDepth := 0
	foundOpening := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if lineNum < startLine {
			continue
		}

		if !capturing {
			capturing = true
		}

		lines = append(lines, line)

		// Count braces to find the end of the block
		for _, ch := range line {
			if ch == '{' {
				braceDepth++
				foundOpening = true
			}
			if ch == '}' {
				braceDepth--
			}
		}

		// Block is complete when we've seen an opening brace and returned to depth 0
		if foundOpening && braceDepth == 0 {
			break
		}

		// Safety: don't read more than 200 lines for a single function
		if len(lines) > 200 {
			lines = append(lines, "    // ... (truncated at 200 lines)")
			break
		}

		// Handle one-liner functions or type declarations without braces
		// (e.g., "type Foo = Bar" or single-line func)
		if lineNum == startLine && !strings.Contains(line, "{") {
			// Check if next line starts a new declaration or is blank
			// For now, capture just this line
			if !strings.HasSuffix(strings.TrimSpace(line), "{") &&
				!strings.HasSuffix(strings.TrimSpace(line), "(") {
				// Might be a single-line declaration — include a few more lines
				// to catch struct fields or interface methods
				continue
			}
		}
	}

	if len(lines) == 0 {
		return "", 0, fmt.Errorf("no content at line %d", startLine)
	}

	endLine := startLine + len(lines) - 1
	return strings.Join(lines, "\n"), endLine, nil
}

// ResolveSourcePath tries to find the actual file on disk.
func ResolveSourcePath(sourceFile, projectDir string) string {
	// Try as-is (relative to project)
	p := filepath.Join(projectDir, sourceFile)
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// Try walking common prefixes
	candidates := []string{
		filepath.Join(projectDir, "src", sourceFile),
		filepath.Join(projectDir, "internal", sourceFile),
		filepath.Join(projectDir, "pkg", sourceFile),
		filepath.Join(projectDir, "cmd", sourceFile),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Walk the project looking for the file by basename
	baseName := filepath.Base(sourceFile)
	var found string
	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == baseName {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
