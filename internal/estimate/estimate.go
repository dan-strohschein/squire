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

	Size              string // TINY, SMALL, MEDIUM, LARGE, XLARGE

	AffectedModules   []string
	ComplexityFactors []string
}

// FromSymbols estimates complexity from explicit symbol names.
func FromSymbols(aidDir string, symbols []string) (*EstimateResult, error) {
	g, err := loader.LoadFromDirectory(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 20)
	return analyze(g, engine, symbols, aidDir)
}

// FromPlan estimates complexity by extracting symbols from a plan file.
func FromPlan(aidDir string, planPath string) (*EstimateResult, error) {
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}

	g, err := loader.LoadFromDirectory(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	matches := ExtractSymbols(string(planBytes), g)
	symbols := make([]string, len(matches))
	for i, m := range matches {
		symbols[i] = m.Name
	}

	engine := query.NewQueryEngine(g, 20)
	return analyze(g, engine, symbols, aidDir)
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
