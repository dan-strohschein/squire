// Package carto wraps the embedded cartograph engine for squire query.
package carto

import (
	"fmt"
	"os"
	"strings"

	"github.com/dan-strohschein/cartograph/pkg/graph"
	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/cartograph/pkg/output"
	"github.com/dan-strohschein/cartograph/pkg/query"
)

// Run executes a cartograph command with the given args.
// This is the embedded equivalent of running the cartograph binary.
func Run(aidDir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	g, err := loader.LoadFromDirectory(aidDir)
	if err != nil {
		return fmt.Errorf("loading AID files: %w", err)
	}

	engine := query.NewQueryEngine(g, 10)
	cmd := args[0]
	remaining := args[1:]

	switch cmd {
	case "stats":
		return runStats(g)

	case "callstack":
		if len(remaining) == 0 {
			return fmt.Errorf("callstack requires a function name")
		}
		funcName := remaining[0]
		direction := query.Both
		for _, arg := range remaining[1:] {
			switch arg {
			case "--up":
				direction = query.Reverse
			case "--down":
				direction = query.Forward
			case "--both":
				direction = query.Both
			}
		}
		result, err := engine.CallStack(funcName, direction)
		if err != nil {
			return err
		}
		output.RenderTree(os.Stdout, result)

	case "depends":
		if len(remaining) == 0 {
			return fmt.Errorf("depends requires a type name")
		}
		result, err := engine.TypeDependents(remaining[0])
		if err != nil {
			return err
		}
		output.RenderTree(os.Stdout, result)

	case "field":
		if len(remaining) == 0 {
			return fmt.Errorf("field requires Type.Field format")
		}
		parts := strings.SplitN(remaining[0], ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("field requires Type.Field format (e.g., User.email)")
		}
		result, err := engine.FieldTouchers(parts[0], parts[1])
		if err != nil {
			return err
		}
		output.RenderTree(os.Stdout, result)

	case "errors":
		if len(remaining) == 0 {
			return fmt.Errorf("errors requires an error type name")
		}
		result, err := engine.ErrorProducers(remaining[0])
		if err != nil {
			return err
		}
		output.RenderTree(os.Stdout, result)

	case "effects":
		if len(remaining) == 0 {
			return fmt.Errorf("effects requires a function name")
		}
		report, err := engine.SideEffects(remaining[0])
		if err != nil {
			return err
		}
		output.RenderEffectTree(os.Stdout, report)

	case "search":
		if len(remaining) == 0 {
			return fmt.Errorf("search requires a pattern")
		}
		var kindFilter graph.NodeKind
		for i, arg := range remaining[1:] {
			if arg == "--kind" && i+1 < len(remaining[1:]) {
				v := remaining[i+2]
				if len(v) > 0 {
					v = strings.ToUpper(v[:1]) + v[1:]
				}
				kindFilter = graph.NodeKind(v)
			}
		}
		result, err := engine.Search(remaining[0], kindFilter)
		if err != nil {
			return err
		}
		renderSearch(result)

	case "list":
		if len(remaining) == 0 {
			return fmt.Errorf("list requires a module name")
		}
		result, err := engine.ListModule(remaining[0])
		if err != nil {
			return err
		}
		renderSearch(result)

	default:
		return fmt.Errorf("unknown query command: %s\nAvailable: callstack, depends, field, errors, effects, search, list, stats", cmd)
	}

	return nil
}

func runStats(g *graph.Graph) error {
	stats := g.Stats()
	fmt.Printf("Graph Statistics\n")
	fmt.Printf("  Nodes: %d\n", stats.NodeCount)
	fmt.Printf("  Edges: %d\n", stats.EdgeCount)
	fmt.Printf("  Modules: %d\n", stats.Modules)
	fmt.Printf("  Nodes by kind:\n")
	for k, v := range stats.NodesByKind {
		fmt.Printf("    %s: %d\n", k, v)
	}
	fmt.Printf("  Edges by kind:\n")
	for k, v := range stats.EdgesByKind {
		fmt.Printf("    %s: %d\n", k, v)
	}
	return nil
}

func renderSearch(result *query.SearchResult) {
	fmt.Printf("Search: %q — %d match(es)\n\n", result.Pattern, result.Total)
	kindOrder := []graph.NodeKind{
		graph.KindModule, graph.KindType, graph.KindTrait,
		graph.KindFunction, graph.KindMethod, graph.KindField,
		graph.KindConstant, graph.KindWorkflow, graph.KindLock,
	}
	for _, kind := range kindOrder {
		nodes, ok := result.Matches[kind]
		if !ok || len(nodes) == 0 {
			continue
		}
		fmt.Printf("  %s (%d):\n", kind, len(nodes))
		for _, n := range nodes {
			loc := ""
			if n.SourceFile != "" {
				loc = fmt.Sprintf(" (%s:%d)", n.SourceFile, n.SourceLine)
			}
			purpose := ""
			if n.Purpose != "" {
				purpose = " — " + n.Purpose
			}
			fmt.Printf("    %s%s%s\n", n.Name, purpose, loc)
		}
		fmt.Println()
	}
}
