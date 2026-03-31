package estimate

import (
	"strings"
	"unicode"

	"github.com/dan-strohschein/cartograph/pkg/graph"
)

// ExtractSymbols finds graph node names mentioned in plan text.
// Returns deduplicated matches sorted by relevance (exact match first, then partial).
func ExtractSymbols(planText string, g *graph.Graph) []SymbolMatch {
	// Build a lookup of all node names in the graph
	nodesByName := map[string]graph.Node{}
	for _, n := range g.AllNodes() {
		if n.Kind == graph.KindModule || n.Kind == graph.KindField {
			continue // skip modules and fields for plan matching
		}
		nodesByName[n.Name] = n
	}

	// Extract candidate words from the plan text
	candidates := extractCandidateWords(planText)

	// Match candidates against graph nodes
	seen := map[string]bool{}
	var matches []SymbolMatch

	for _, word := range candidates {
		if seen[word] {
			continue
		}

		if node, ok := nodesByName[word]; ok {
			seen[word] = true
			matches = append(matches, SymbolMatch{
				Name:   node.Name,
				Kind:   string(node.Kind),
				Module: node.Module,
			})
		}
	}

	return matches
}

// extractCandidateWords pulls potential symbol names from text.
// Looks for capitalized words, dotted names (Type.Method), and backtick-quoted identifiers.
func extractCandidateWords(text string) []string {
	var candidates []string
	seen := map[string]bool{}

	add := func(word string) {
		word = strings.TrimSpace(word)
		word = strings.Trim(word, ".,;:(){}[]\"'`")
		if word != "" && !seen[word] && !isCommonWord(word) {
			seen[word] = true
			candidates = append(candidates, word)
		}
	}

	// Extract backtick-quoted identifiers first (highest confidence)
	for _, part := range strings.Split(text, "`") {
		// Every other part is inside backticks
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && isLikelySymbol(trimmed) {
			add(trimmed)
		}
	}

	// Extract words that look like symbols
	words := strings.Fields(text)
	for _, word := range words {
		clean := strings.Trim(word, ".,;:(){}[]\"'`")

		// Dotted names: Type.Method, Type.Field
		if strings.Contains(clean, ".") {
			add(clean)
			parts := strings.Split(clean, ".")
			for _, p := range parts {
				if isLikelySymbol(p) {
					add(p)
				}
			}
			continue
		}

		// Capitalized words (likely Go exported names)
		if len(clean) > 1 && unicode.IsUpper(rune(clean[0])) && isLikelySymbol(clean) {
			add(clean)
		}

		// CamelCase or PascalCase words
		if hasMixedCase(clean) && isLikelySymbol(clean) {
			add(clean)
		}
	}

	return candidates
}

func isLikelySymbol(s string) bool {
	if len(s) < 2 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func hasMixedCase(s string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range s {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	return hasUpper && hasLower
}

// isCommonWord filters out English words that aren't code symbols.
var commonWords = map[string]bool{
	"The": true, "This": true, "That": true, "These": true, "Those": true,
	"When": true, "Where": true, "Which": true, "What": true, "Who": true,
	"How": true, "Why": true, "Each": true, "Every": true, "All": true,
	"Some": true, "Any": true, "Many": true, "Most": true, "Much": true,
	"Add": true, "Remove": true, "Change": true, "Update": true, "Create": true,
	"Delete": true, "Move": true, "Rename": true, "Make": true, "Use": true,
	"For": true, "From": true, "Into": true, "With": true, "Without": true,
	"Before": true, "After": true, "Between": true, "Through": true,
	"Should": true, "Would": true, "Could": true, "Must": true, "Will": true,
	"Also": true, "Then": true, "Just": true, "Only": true, "Still": true,
	"Plan": true, "Step": true, "Phase": true, "File": true, "Files": true,
	"Line": true, "Lines": true, "Code": true, "Note": true, "TODO": true,
	"See": true, "Run": true, "Test": true, "Build": true, "Check": true,
	"New": true, "Old": true, "First": true, "Last": true, "Next": true,
	"True": true, "False": true, "None": true, "Null": true, "Nil": true,
	"Int": true, "String": true, "Bool": true, "Error": true, "Context": true,
	"Returns": true, "Takes": true, "Calls": true, "Called": true,
	"Currently": true, "Instead": true, "Because": true, "Since": true,
	"However": true, "Although": true, "Already": true,
	"Package": true, "Module": true, "Import": true, "Type": true,
	"Function": true, "Method": true, "Interface": true, "Struct": true,
}

func isCommonWord(s string) bool {
	return commonWords[s]
}
