// Package excerpt extracts specific function/type bodies from source files,
// filtered by symbol name. File-centric complement to show (which is symbol-centric).
package excerpt

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/squire/internal/show"
)

// FileSpec describes a file and optional symbol filter.
type FileSpec struct {
	Path    string
	Symbols []string // empty = all declarations
}

// ExcerptResult holds extracted source for one symbol within a file.
type ExcerptResult struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Source    string
	Purpose   string
	Signature string
}

// FileExcerpt holds results for a single file.
type FileExcerpt struct {
	FilePath   string
	TotalDecls int            // total declarations found in the file
	Symbols    []ExcerptResult // extracted symbols in file order
	Skipped    []string        // requested symbols not found
}

// ParseSpecs parses CLI args like "service.go:Create,retryOne" into FileSpec slices.
func ParseSpecs(args []string) []FileSpec {
	var specs []FileSpec
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		parts := strings.SplitN(arg, ":", 2)
		spec := FileSpec{Path: parts[0]}
		if len(parts) == 2 && parts[1] != "" {
			spec.Symbols = strings.Split(parts[1], ",")
		}
		specs = append(specs, spec)
	}
	return specs
}

// FromFile extracts specific symbols from a file.
// If symbols is empty, extracts all top-level declarations.
// If aidDir is non-empty, enriches results with AID metadata.
func FromFile(filePath string, symbols []string, aidDir, projectDir string) (*FileExcerpt, error) {
	// Resolve the actual file path
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(projectDir, filePath)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	result := &FileExcerpt{FilePath: filePath}

	// Try graph-based extraction first
	if aidDir != "" {
		g, err := loader.LoadFromDirectoryCached(aidDir)
		if err == nil {
			return fromFileWithGraph(absPath, filePath, symbols, g, projectDir, result)
		}
	}

	// Fallback: scan source for declarations
	return fromFileWithScan(absPath, filePath, symbols, result)
}

// FromMultipleFiles handles multiple file specs.
func FromMultipleFiles(specs []FileSpec, aidDir, projectDir string) ([]FileExcerpt, error) {
	var results []FileExcerpt
	var g *graph.Graph

	// Load graph once if available
	if aidDir != "" {
		var err error
		g, err = loader.LoadFromDirectoryCached(aidDir)
		if err != nil {
			g = nil // proceed without graph
		}
	}

	for _, spec := range specs {
		absPath := spec.Path
		if !filepath.IsAbs(spec.Path) {
			absPath = filepath.Join(projectDir, spec.Path)
		}
		if _, err := os.Stat(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: file not found: %s\n", spec.Path)
			continue
		}

		fe := &FileExcerpt{FilePath: spec.Path}
		var err error
		if g != nil {
			fe, err = fromFileWithGraph(absPath, spec.Path, spec.Symbols, g, projectDir, fe)
		} else {
			fe, err = fromFileWithScan(absPath, spec.Path, spec.Symbols, fe)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			continue
		}
		results = append(results, *fe)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no files could be processed")
	}
	return results, nil
}

func fromFileWithGraph(absPath, relPath string, symbols []string, g *graph.Graph, projectDir string, result *FileExcerpt) (*FileExcerpt, error) {
	// Find all nodes in the graph that reference this file
	var fileNodes []graph.Node
	baseName := filepath.Base(relPath)

	for _, n := range g.AllNodes() {
		if n.Kind == graph.KindField || n.Kind == graph.KindModule || n.Kind == graph.KindConstant {
			continue
		}
		if n.SourceFile == "" || n.SourceLine == 0 {
			continue
		}
		// Match by basename or full relative path
		if filepath.Base(n.SourceFile) == baseName || n.SourceFile == relPath {
			// Verify the resolved path actually matches our target file
			resolved := show.ResolveSourcePath(n.SourceFile, projectDir)
			if resolved == absPath {
				fileNodes = append(fileNodes, n)
			}
		}
	}

	result.TotalDecls = len(fileNodes)

	if len(fileNodes) == 0 {
		// No graph nodes for this file — fall back to scan
		return fromFileWithScan(absPath, relPath, symbols, result)
	}

	// Filter to requested symbols
	symbolSet := make(map[string]bool)
	for _, s := range symbols {
		symbolSet[s] = true
	}

	var matched []graph.Node
	for _, n := range fileNodes {
		if len(symbols) == 0 {
			matched = append(matched, n)
			continue
		}
		// Check exact name match and suffix match (Type.Method → Method)
		if symbolSet[n.Name] {
			matched = append(matched, n)
			continue
		}
		parts := strings.Split(n.Name, ".")
		if len(parts) > 1 && symbolSet[parts[len(parts)-1]] {
			matched = append(matched, n)
			continue
		}
	}

	// Track which requested symbols were not found
	if len(symbols) > 0 {
		foundSymbols := make(map[string]bool)
		for _, n := range matched {
			foundSymbols[n.Name] = true
			parts := strings.Split(n.Name, ".")
			if len(parts) > 1 {
				foundSymbols[parts[len(parts)-1]] = true
			}
		}
		for _, s := range symbols {
			if !foundSymbols[s] {
				result.Skipped = append(result.Skipped, s)
			}
		}
	}

	// Sort by source line
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].SourceLine < matched[j].SourceLine
	})

	// Extract source for each matched node
	for _, n := range matched {
		source, endLine, err := show.ExtractBlock(absPath, n.SourceLine)
		if err != nil {
			continue
		}
		result.Symbols = append(result.Symbols, ExcerptResult{
			Name:      n.Name,
			Kind:      string(n.Kind),
			StartLine: n.SourceLine,
			EndLine:   endLine,
			Source:    source,
			Purpose:   n.Purpose,
			Signature: n.Signature,
		})
	}

	return result, nil
}

func fromFileWithScan(absPath, relPath string, symbols []string, result *FileExcerpt) (*FileExcerpt, error) {
	// Scan file for top-level func/type declarations
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type decl struct {
		name string
		kind string
		line int
	}
	var decls []decl

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect top-level func declarations
		if strings.HasPrefix(trimmed, "func ") {
			name := extractFuncName(trimmed)
			if name != "" {
				decls = append(decls, decl{name: name, kind: "Function", line: lineNum})
			}
		}

		// Detect top-level type declarations
		if strings.HasPrefix(trimmed, "type ") {
			name := extractTypeName(trimmed)
			if name != "" {
				decls = append(decls, decl{name: name, kind: "Type", line: lineNum})
			}
		}
	}

	result.TotalDecls = len(decls)

	// Filter to requested symbols
	symbolSet := make(map[string]bool)
	for _, s := range symbols {
		symbolSet[s] = true
	}

	var toExtract []decl
	for _, d := range decls {
		if len(symbols) == 0 || symbolSet[d.name] {
			toExtract = append(toExtract, d)
		}
	}

	// Track skipped
	if len(symbols) > 0 {
		foundSymbols := make(map[string]bool)
		for _, d := range toExtract {
			foundSymbols[d.name] = true
		}
		for _, s := range symbols {
			if !foundSymbols[s] {
				result.Skipped = append(result.Skipped, s)
			}
		}
	}

	// Extract source blocks
	for _, d := range toExtract {
		source, endLine, err := show.ExtractBlock(absPath, d.line)
		if err != nil {
			continue
		}
		result.Symbols = append(result.Symbols, ExcerptResult{
			Name:      d.name,
			Kind:      d.kind,
			StartLine: d.line,
			EndLine:   endLine,
			Source:    source,
		})
	}

	return result, nil
}

// extractFuncName pulls the function name from a "func" line.
// Handles: "func Name(...", "func (r *Recv) Name(..."
func extractFuncName(line string) string {
	line = strings.TrimPrefix(line, "func ")

	// Method: "(r *Recv) Name(...)"
	if strings.HasPrefix(line, "(") {
		idx := strings.Index(line, ")")
		if idx < 0 {
			return ""
		}
		line = strings.TrimSpace(line[idx+1:])
	}

	// Extract name up to first non-identifier char
	var name []rune
	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			name = append(name, r)
		} else {
			break
		}
	}
	if len(name) == 0 {
		return ""
	}
	return string(name)
}

// extractTypeName pulls the type name from a "type" line.
func extractTypeName(line string) string {
	line = strings.TrimPrefix(line, "type ")
	var name []rune
	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			name = append(name, r)
		} else {
			break
		}
	}
	if len(name) == 0 {
		return ""
	}
	return string(name)
}

// Format renders file excerpts in compact, agent-optimized format.
func Format(excerpts []FileExcerpt) string {
	var b strings.Builder

	for i, fe := range excerpts {
		if i > 0 {
			b.WriteString("\n")
		}

		// File header
		if fe.TotalDecls > 0 {
			fmt.Fprintf(&b, "// %s — %d of %d symbols\n", fe.FilePath, len(fe.Symbols), fe.TotalDecls)
		} else {
			fmt.Fprintf(&b, "// %s\n", fe.FilePath)
		}

		for j, sym := range fe.Symbols {
			if j > 0 {
				b.WriteString("\n// ---\n")
			}
			b.WriteString("\n")

			// Purpose comment
			if sym.Purpose != "" {
				fmt.Fprintf(&b, "// %s\n", sym.Purpose)
			}

			// Source code
			b.WriteString(sym.Source)
			b.WriteString("\n")

			// Location footer
			fmt.Fprintf(&b, "// %s:%d-%d\n", fe.FilePath, sym.StartLine, sym.EndLine)
		}

		// Skipped symbols
		if len(fe.Skipped) > 0 {
			fmt.Fprintf(&b, "\n// Not found: %s\n", strings.Join(fe.Skipped, ", "))
		}
	}

	return b.String()
}
