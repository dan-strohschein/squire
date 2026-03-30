// Package refactor wraps chisel's refactoring engine for squire.
package refactor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dan-strohschein/cartograph/pkg/loader"
	"github.com/dan-strohschein/cartograph/pkg/output"
	"github.com/dan-strohschein/cartograph/pkg/query"
	"github.com/dan-strohschein/chisel/resolve"
)

// EmbeddedGraphQuerier uses the embedded cartograph engine instead of
// shelling out to a binary. Implements resolve.GraphQuerier.
type EmbeddedGraphQuerier struct{}

func (e *EmbeddedGraphQuerier) Query(command string, args []string, aidDir string) (*resolve.GraphResult, error) {
	if aidDir == "" {
		return nil, fmt.Errorf("no .aidocs/ directory specified")
	}

	g, err := loader.LoadFromDirectory(aidDir)
	if err != nil {
		return nil, fmt.Errorf("loading graph: %w", err)
	}

	engine := query.NewQueryEngine(g, 10)

	switch command {
	case "callstack":
		if len(args) == 0 {
			return nil, fmt.Errorf("callstack requires a function name")
		}
		direction := query.Both
		for _, a := range args[1:] {
			switch a {
			case "--up":
				direction = query.Reverse
			case "--down":
				direction = query.Forward
			}
		}
		qr, err := engine.CallStack(args[0], direction)
		if err != nil {
			return nil, err
		}
		return queryResultToGraphResult(qr)

	case "depends":
		if len(args) == 0 {
			return nil, fmt.Errorf("depends requires a type name")
		}
		qr, err := engine.TypeDependents(args[0])
		if err != nil {
			return nil, err
		}
		return queryResultToGraphResult(qr)

	case "field":
		if len(args) == 0 {
			return nil, fmt.Errorf("field requires a field name")
		}
		parts := strings.SplitN(args[0], ".", 2)
		if len(parts) == 2 {
			qr, err := engine.FieldTouchers(parts[0], parts[1])
			if err != nil {
				return nil, err
			}
			return queryResultToGraphResult(qr)
		}
		// No dot — try depends as fallback (field name might be a type)
		qr, err := engine.TypeDependents(args[0])
		if err != nil {
			// Return empty result instead of error — chisel handles empty gracefully
			return &resolve.GraphResult{
				Query:   fmt.Sprintf("Field(%s)", args[0]),
				Summary: "No results",
			}, nil
		}
		return queryResultToGraphResult(qr)

	case "errors":
		if len(args) == 0 {
			return nil, fmt.Errorf("errors requires an error type")
		}
		qr, err := engine.ErrorProducers(args[0])
		if err != nil {
			return nil, err
		}
		return queryResultToGraphResult(qr)

	case "effects":
		if len(args) == 0 {
			return nil, fmt.Errorf("effects requires a function name")
		}
		_, err := engine.SideEffects(args[0])
		if err != nil {
			return nil, err
		}
		return &resolve.GraphResult{
			Query:   fmt.Sprintf("Effects(%s)", args[0]),
			Summary: fmt.Sprintf("Effects analysis for %s", args[0]),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported query command: %s", command)
	}
}

// queryResultToGraphResult converts via JSON round-trip.
// Both types have identical JSON structure.
func queryResultToGraphResult(qr *query.QueryResult) (*resolve.GraphResult, error) {
	var buf bytes.Buffer
	output.RenderJSON(&buf, qr)

	var result resolve.GraphResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("converting result: %w", err)
	}
	return &result, nil
}
