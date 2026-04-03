package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/cartograph/pkg/query"
	"github.com/dan-strohschein/squire/internal/estimate"
)

// AnchoredFinding is a Finding cross-referenced against the AID graph.
type AnchoredFinding struct {
	Finding
	GraphNode string // matched node name from graph
	GraphKind string // Function, Type, etc.
	Module    string // AID module
	Purpose   string // from AID @purpose
	Callers   []string
	Callees   []string
}

// DigestResult is the compressed output.
type DigestResult struct {
	Task          string
	Anchored      []AnchoredFinding
	Unanchored    []Finding
	AffectedPkgs  []string
	OpenQuestions []string
}

// Digest reads a findings file, cross-references against AID, and produces
// a compressed summary.
func Digest(findingsPath, aidDir string, taskLabel string) (*DigestResult, error) {
	content, err := os.ReadFile(findingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading findings: %w", err)
	}

	findings, openQuestions := ParseFindings(string(content))

	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 1)

	// Extract task label from first heading if not provided
	if taskLabel == "" {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				taskLabel = strings.TrimPrefix(line, "# ")
				break
			}
		}
	}

	result := &DigestResult{
		Task:          taskLabel,
		OpenQuestions: openQuestions,
	}

	pkgSet := map[string]bool{}

	for _, f := range findings {
		anchored := tryAnchor(f, g, engine)
		if anchored != nil {
			result.Anchored = append(result.Anchored, *anchored)
			if anchored.Module != "" {
				pkgSet[anchored.Module] = true
			}
		} else if f.RawText != "" {
			// Try to find symbols mentioned in the text
			matches := estimate.ExtractSymbols(f.RawText, g)
			if len(matches) > 0 {
				// Anchor to the first matched symbol
				node := findNode(g, matches[0].Name)
				if node != nil {
					af := anchorToNode(f, *node, engine)
					result.Anchored = append(result.Anchored, af)
					if af.Module != "" {
						pkgSet[af.Module] = true
					}
					continue
				}
			}
			result.Unanchored = append(result.Unanchored, f)
		}
	}

	for pkg := range pkgSet {
		result.AffectedPkgs = append(result.AffectedPkgs, pkg)
	}
	sort.Strings(result.AffectedPkgs)

	return result, nil
}

func tryAnchor(f Finding, g *graph.Graph, engine *query.QueryEngine) *AnchoredFinding {
	// Strategy 1: Match by file + line number
	if f.File != "" && f.LineStart > 0 {
		for _, n := range g.AllNodes() {
			if n.Kind == graph.KindField || n.Kind == graph.KindModule || n.Kind == graph.KindConstant {
				continue
			}
			if n.SourceFile == "" || n.SourceLine == 0 {
				continue
			}
			// Match by basename
			if filepath.Base(n.SourceFile) != filepath.Base(f.File) {
				continue
			}
			// Within 5 lines of the finding's line reference
			diff := n.SourceLine - f.LineStart
			if diff < 0 {
				diff = -diff
			}
			if diff <= 5 {
				af := anchorToNode(f, n, engine)
				return &af
			}
		}
	}

	// Strategy 2: Match by symbol name
	if f.SymbolName != "" {
		nodes := g.NodesByName(f.SymbolName)
		if len(nodes) == 0 {
			// Try suffix match (e.g., "ListByUserID" matching "devicesRepo.ListByUserID")
			for _, n := range g.AllNodes() {
				if n.Kind == graph.KindField || n.Kind == graph.KindModule {
					continue
				}
				if strings.HasSuffix(n.Name, "."+f.SymbolName) || strings.HasSuffix(n.Name, f.SymbolName) {
					nodes = append(nodes, n)
				}
			}
		}
		if len(nodes) > 0 {
			af := anchorToNode(f, nodes[0], engine)
			return &af
		}
	}

	return nil
}

func anchorToNode(f Finding, node graph.Node, engine *query.QueryEngine) AnchoredFinding {
	af := AnchoredFinding{
		Finding:   f,
		GraphNode: node.Name,
		GraphKind: string(node.Kind),
		Module:    node.Module,
		Purpose:   node.Purpose,
	}

	// Get callers (1 hop up)
	if result, err := engine.CallStack(node.Name, query.Reverse); err == nil && result != nil {
		seen := map[string]bool{}
		for _, path := range result.Paths {
			if len(path.Nodes) > 1 {
				name := path.Nodes[1].Name
				if !seen[name] {
					seen[name] = true
					af.Callers = append(af.Callers, name)
					if len(af.Callers) >= 5 {
						break
					}
				}
			}
		}
	}

	// Get callees (1 hop down)
	if result, err := engine.CallStack(node.Name, query.Forward); err == nil && result != nil {
		seen := map[string]bool{}
		for _, path := range result.Paths {
			if len(path.Nodes) > 1 {
				name := path.Nodes[1].Name
				if !seen[name] {
					seen[name] = true
					af.Callees = append(af.Callees, name)
					if len(af.Callees) >= 5 {
						break
					}
				}
			}
		}
	}

	return af
}

func findNode(g *graph.Graph, name string) *graph.Node {
	nodes := g.NodesByName(name)
	if len(nodes) > 0 {
		return &nodes[0]
	}
	return nil
}

// Format renders a DigestResult as compact markdown.
func Format(result *DigestResult) string {
	var b strings.Builder

	// Title
	if result.Task != "" {
		fmt.Fprintf(&b, "# Digest: %s\n", result.Task)
	} else {
		b.WriteString("# Digest\n")
	}

	// Affected packages
	if len(result.AffectedPkgs) > 0 {
		fmt.Fprintf(&b, "## Affected: %s\n", strings.Join(result.AffectedPkgs, ", "))
	}
	b.WriteString("\n")

	// Anchored findings
	for _, af := range result.Anchored {
		fmt.Fprintf(&b, "### %s (%s, %s)\n", af.GraphNode, af.GraphKind, af.Module)
		if af.Purpose != "" {
			fmt.Fprintf(&b, "Purpose: %s\n", af.Purpose)
		}
		if af.RawText != "" {
			fmt.Fprintf(&b, "Finding: %s\n", compactText(af.RawText))
		}
		if af.File != "" {
			if af.LineEnd > 0 {
				fmt.Fprintf(&b, "Lines: %s:%d-%d\n", af.File, af.LineStart, af.LineEnd)
			} else if af.LineStart > 0 {
				fmt.Fprintf(&b, "Lines: %s:%d\n", af.File, af.LineStart)
			}
		}
		if len(af.Callers) > 0 {
			fmt.Fprintf(&b, "Callers: %s\n", strings.Join(af.Callers, ", "))
		}
		if len(af.Callees) > 0 {
			fmt.Fprintf(&b, "Callees: %s\n", strings.Join(af.Callees, ", "))
		}
		b.WriteString("\n")
	}

	// Unanchored findings
	if len(result.Unanchored) > 0 {
		b.WriteString("## Unanchored\n")
		for _, f := range result.Unanchored {
			text := compactText(f.RawText)
			if text != "" {
				fmt.Fprintf(&b, "- %s\n", text)
			}
		}
		b.WriteString("\n")
	}

	// Open questions
	if len(result.OpenQuestions) > 0 {
		b.WriteString("## Open Questions\n")
		for _, q := range result.OpenQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// compactText collapses multi-line text into a single line,
// trimming excess whitespace.
func compactText(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var parts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "?") && !strings.HasPrefix(trimmed, "TODO") {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}
