// Package digest compresses session findings into compact AID-anchored summaries.
package digest

import (
	"regexp"
	"strings"
)

// Finding represents one observation from a session.
type Finding struct {
	File       string // source file referenced, if any
	LineStart  int    // 0 if not specified
	LineEnd    int    // 0 if not specified
	SymbolName string // extracted from header or matched from graph
	RawText    string // the observation text
}

var fileLineRe = regexp.MustCompile(`(\S+\.\w+):(\d+)(?:-(\d+))?`)

// ParseFindings extracts structured findings from freeform markdown.
// Expects sections separated by "## " headers. Each header may contain
// a file:line reference and/or a symbol name after " - ".
func ParseFindings(content string) (findings []Finding, openQuestions []string) {
	// Split on ## headers
	sections := strings.Split("\n"+content, "\n## ")

	for i, section := range sections {
		if i == 0 {
			// Content before first ## header — check for top-level title
			continue
		}

		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// Split header from body
		lines := strings.SplitN(section, "\n", 2)
		header := strings.TrimSpace(lines[0])
		body := ""
		if len(lines) > 1 {
			body = strings.TrimSpace(lines[1])
		}

		// Extract open questions from body
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "?") || strings.HasPrefix(trimmed, "TODO") {
				openQuestions = append(openQuestions, trimmed)
			}
		}

		// Check if this is an "Observation" or similar non-file section
		lowerHeader := strings.ToLower(header)
		if lowerHeader == "observation" || lowerHeader == "observations" ||
			lowerHeader == "notes" || lowerHeader == "summary" {
			findings = append(findings, Finding{
				RawText: body,
			})
			continue
		}

		f := Finding{RawText: body}

		// Extract file:line from header
		if match := fileLineRe.FindStringSubmatch(header); match != nil {
			f.File = match[1]
			f.LineStart = atoi(match[2])
			if match[3] != "" {
				f.LineEnd = atoi(match[3])
			}
		}

		// Extract symbol name after " - "
		if idx := strings.Index(header, " - "); idx >= 0 {
			f.SymbolName = strings.TrimSpace(header[idx+3:])
		} else if f.File == "" {
			// No file reference and no " - " separator — treat the whole header as a potential symbol
			f.SymbolName = header
		}

		findings = append(findings, f)
	}

	return findings, openQuestions
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
