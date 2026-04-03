// Package estimate analyzes implementation plans against the semantic graph
// to produce story point estimates.
package estimate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/cartograph/pkg/query"
)

// SymbolMatch is a symbol from the plan that was found in the graph.
type SymbolMatch struct {
	Name   string
	Kind   string
	Module string
}

// EstimateResult holds the complete estimation output.
type EstimateResult struct {
	Symbols           []SymbolMatch
	UnmatchedSymbols  []string // symbols mentioned in plan but not in graph

	Files             int
	Functions         int
	Packages          int
	TestFiles         int

	Locks             int
	ErrorMaps         int
	Antipatterns      int
	Invariants        int

	Size              string // TINY, SMALL, MEDIUM, LARGE, XLARGE, or UNCLEAR

	AffectedModules   []string
	ComplexityFactors []string

	// Suggestions are offered when the plan is too vague to estimate.
	Unclear           bool
	Suggestions       []string
}

// FromSymbols estimates complexity from explicit symbol names.
func FromSymbols(aidDir string, symbols []string) (*EstimateResult, error) {
	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 20)
	result, err := analyze(g, engine, symbols, aidDir)
	if err != nil {
		return nil, err
	}

	// If all symbols were unmatched, mark as unclear
	if len(result.Symbols) == 0 && len(result.UnmatchedSymbols) > 0 {
		result.Unclear = true
		result.Size = "UNCLEAR"
		// Suggest similar symbols from the graph
		for _, sym := range result.UnmatchedSymbols {
			sr, searchErr := engine.Search("*"+sym+"*", "")
			if searchErr != nil || sr.Total == 0 {
				continue
			}
			seen := map[string]bool{}
			for _, nodes := range sr.Matches {
				for _, n := range nodes {
					if n.Kind == graph.KindModule || n.Kind == graph.KindField || n.Kind == graph.KindConstant {
						continue
					}
					if !seen[n.Name] && len(result.Suggestions) < 10 {
						seen[n.Name] = true
						result.Suggestions = append(result.Suggestions, n.Name)
					}
				}
			}
		}
	}

	return result, nil
}

// FromPlan estimates complexity by extracting symbols from a plan file.
// If no symbols are found, returns an UNCLEAR result with suggestions.
func FromPlan(aidDir string, planPath string) (*EstimateResult, error) {
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}

	g, err := loader.LoadFromDirectoryCached(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	matches := ExtractSymbols(string(planBytes), g)

	if len(matches) == 0 {
		return buildUnclearResult(string(planBytes), g), nil
	}

	symbols := make([]string, len(matches))
	for i, m := range matches {
		symbols[i] = m.Name
	}

	engine := query.NewQueryEngine(g, 20)
	return analyze(g, engine, symbols, aidDir)
}

// buildUnclearResult is returned when a plan is too vague to estimate.
// It extracts keywords from the plan and suggests matching graph symbols
// so the user can refine the plan.
func buildUnclearResult(planText string, g *graph.Graph) *EstimateResult {
	result := &EstimateResult{
		Unclear: true,
		Size:    "UNCLEAR",
	}

	// Extract keywords that might be domain terms (even if not exact symbol matches)
	keywords := extractKeywords(planText)

	// For each keyword, search the graph for partial matches
	engine := query.NewQueryEngine(g, 10)
	seen := map[string]bool{}

	for _, kw := range keywords {
		sr, err := engine.Search("*"+kw+"*", "")
		if err != nil || sr.Total == 0 {
			continue
		}
		// Collect up to 3 suggestions per keyword
		count := 0
		for _, nodes := range sr.Matches {
			for _, n := range nodes {
				if n.Kind == graph.KindModule || n.Kind == graph.KindField || n.Kind == graph.KindConstant {
					continue
				}
				if !seen[n.Name] {
					seen[n.Name] = true
					result.Suggestions = append(result.Suggestions, n.Name)
					count++
				}
				if count >= 3 {
					break
				}
			}
			if count >= 3 {
				break
			}
		}
	}

	// Cap total suggestions
	if len(result.Suggestions) > 10 {
		result.Suggestions = result.Suggestions[:10]
	}

	return result
}

// extractKeywords pulls potential domain terms from plan text.
// These are words that might partially match symbol names.
func extractKeywords(text string) []string {
	var keywords []string
	seen := map[string]bool{}

	for _, word := range strings.Fields(text) {
		clean := strings.Trim(word, ".,;:(){}[]\"'`-")
		clean = strings.ToLower(clean)

		if len(clean) < 4 {
			continue
		}
		if stopWords[clean] {
			continue
		}
		if !seen[clean] {
			seen[clean] = true
			keywords = append(keywords, clean)
		}
	}
	return keywords
}

var stopWords = map[string]bool{
	"this": true, "that": true, "these": true, "those": true, "with": true,
	"from": true, "into": true, "will": true, "would": true, "could": true,
	"should": true, "must": true, "have": true, "been": true, "being": true,
	"were": true, "they": true, "them": true, "their": true, "what": true,
	"when": true, "where": true, "which": true, "there": true, "then": true,
	"than": true, "each": true, "every": true, "some": true, "also": true,
	"just": true, "only": true, "still": true, "about": true, "after": true,
	"before": true, "between": true, "through": true, "during": true,
	"make": true, "need": true, "want": true, "change": true, "update": true,
	"refactor": true, "clean": true, "improve": true, "implement": true,
	"file": true, "files": true, "code": true, "plan": true, "step": true,
	"line": true, "lines": true, "note": true, "todo": true, "like": true,
	"more": true, "most": true, "much": true, "many": true, "very": true,
	"because": true, "since": true, "however": true, "although": true,
	"currently": true, "instead": true, "already": true, "other": true,
}

func analyze(g *graph.Graph, engine *query.QueryEngine, symbols []string, aidDir string) (*EstimateResult, error) {
	result := &EstimateResult{}

	affectedNodes := map[graph.NodeID]graph.Node{}
	affectedFiles := map[string]bool{}
	affectedModules := map[string]bool{}

	for _, sym := range symbols {
		// Try to find the symbol in the graph
		nodes := g.NodesByName(sym)
		if len(nodes) == 0 {
			result.UnmatchedSymbols = append(result.UnmatchedSymbols, sym)
			continue
		}

		node := nodes[0]
		result.Symbols = append(result.Symbols, SymbolMatch{
			Name:   node.Name,
			Kind:   string(node.Kind),
			Module: node.Module,
		})

		// Collect the node itself
		affectedNodes[node.ID] = node
		if node.SourceFile != "" {
			affectedFiles[node.SourceFile] = true
		}
		affectedModules[node.Module] = true

		// Run appropriate fan-out query based on node kind
		switch node.Kind {
		case graph.KindType, graph.KindTrait:
			qr, err := engine.TypeDependents(sym)
			if err == nil {
				collectFromResult(qr, affectedNodes, affectedFiles, affectedModules)
			}

		case graph.KindFunction, graph.KindMethod:
			// Callers (upstream impact)
			qr, err := engine.CallStack(sym, query.Reverse)
			if err == nil {
				collectFromResult(qr, affectedNodes, affectedFiles, affectedModules)
			}
			// Callees (downstream for understanding, not counted as "to change")
			// We don't add these to affected counts — just the callers matter for effort

		case graph.KindField:
			// Try to parse as Type.Field
			parts := strings.SplitN(sym, ".", 2)
			if len(parts) == 2 {
				qr, err := engine.FieldTouchers(parts[0], parts[1])
				if err == nil {
					collectFromResult(qr, affectedNodes, affectedFiles, affectedModules)
				}
			}
		}
	}

	// Count affected items
	result.Functions = 0
	for _, n := range affectedNodes {
		if n.Kind == graph.KindFunction || n.Kind == graph.KindMethod {
			result.Functions++
		}
	}
	result.Files = len(affectedFiles)
	result.Packages = len(affectedModules)

	// Count test files
	for f := range affectedFiles {
		if strings.HasSuffix(f, "_test.go") || strings.Contains(f, "/test") || strings.Contains(f, "_test/") {
			result.TestFiles++
		}
	}

	// Build affected modules list
	for mod := range affectedModules {
		result.AffectedModules = append(result.AffectedModules, mod)
	}

	// Scan annotations in affected modules
	scanAnnotations(aidDir, affectedModules, result)

	// Compute size
	result.Size = ComputeSize(result)

	return result, nil
}

func collectFromResult(qr *query.QueryResult, nodes map[graph.NodeID]graph.Node, files map[string]bool, modules map[string]bool) {
	for _, path := range qr.Paths {
		for _, n := range path.Nodes {
			nodes[n.ID] = n
			if n.SourceFile != "" {
				files[n.SourceFile] = true
			}
			modules[n.Module] = true
		}
	}
}

func scanAnnotations(aidDir string, modules map[string]bool, result *EstimateResult) {
	aids, _ := filepath.Glob(filepath.Join(aidDir, "*.aid"))
	for _, aidPath := range aids {
		af, _, err := parser.ParseFile(aidPath)
		if err != nil {
			continue
		}

		// Check if this AID file's module is in our affected set
		if !modules[af.Header.Module] {
			// Also check by filename match
			base := strings.TrimSuffix(filepath.Base(aidPath), ".aid")
			found := false
			for mod := range modules {
				if mod == base || strings.HasSuffix(mod, "/"+base) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		for _, ann := range af.Annotations {
			switch ann.Kind {
			case "lock":
				result.Locks++
				name := ann.Name
				if name == "" {
					name = "unnamed lock"
				}
				purpose := ""
				if f, ok := ann.Fields["purpose"]; ok {
					purpose = " — " + f.Value()
				}
				result.ComplexityFactors = append(result.ComplexityFactors,
					fmt.Sprintf("@lock %s in affected code%s", name, purpose))

			case "error_map":
				result.ErrorMaps++
				name := ann.Name
				if name == "" {
					name = "error map"
				}
				result.ComplexityFactors = append(result.ComplexityFactors,
					fmt.Sprintf("@error_map %s — may need new error variants", name))

			case "antipattern", "antipatterns":
				result.Antipatterns++

			case "invariant", "invariants":
				result.Invariants++
				if f, ok := ann.Fields[ann.Kind]; ok {
					val := f.Value()
					if len(val) > 80 {
						val = val[:80] + "..."
					}
					result.ComplexityFactors = append(result.ComplexityFactors,
						fmt.Sprintf("@invariant: %s", val))
				}
			}
		}
	}

	if result.Antipatterns > 0 {
		result.ComplexityFactors = append(result.ComplexityFactors,
			fmt.Sprintf("%d @antipattern(s) in affected code — review before changing", result.Antipatterns))
	}
}
