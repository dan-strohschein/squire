package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/squire/internal/detect"
	"github.com/dan-strohschein/squire/internal/generate"
)

// Generate creates the AGENTS.md file in .aidocs/.
func Generate(project *detect.Project, stats *generate.GraphStats) error {
	if err := os.MkdirAll(project.AidDir, 0755); err != nil {
		return err
	}

	var b strings.Builder

	projectName := filepath.Base(project.Module)

	b.WriteString(fmt.Sprintf("# AI Agent Guide — %s\n\n", projectName))
	b.WriteString("This project uses AID (Agent Interface Documents) for structured code documentation.\n")
	b.WriteString("AID files reduce the code you need to read by documenting function signatures, call\n")
	b.WriteString("graphs, workflows, invariants, and failure modes.\n\n")

	// How to Use
	b.WriteString("## How to Use\n\n")
	b.WriteString("1. **Start with the manifest**: Read `.aidocs/manifest.aid` for the full package map\n")
	b.WriteString("2. **Read AID before source**: For any package, read its `.aid` file first — it has\n")
	b.WriteString("   function signatures (@fn), call graphs (@calls), workflows (@workflow), and known\n")
	b.WriteString("   issues (@antipatterns)\n")
	b.WriteString("3. **Query the graph**: Use `squire query` to trace dependencies without reading source:\n")
	b.WriteString("   - `squire query callstack <function> --up` — who calls this?\n")
	b.WriteString("   - `squire query callstack <function> --down` — what does this call?\n")
	b.WriteString("   - `squire query depends <Type>` — what depends on this type?\n")
	b.WriteString("   - `squire query search <pattern>` — find by name\n")
	b.WriteString("4. **Refactor with precision**: Use `squire refactor` for codebase-wide changes:\n")
	b.WriteString("   - `squire refactor rename <old> <new>` — rename across all files\n")
	b.WriteString("   - `squire refactor move <symbol> <dest>` — move between packages\n")
	b.WriteString("   - `squire refactor propagate <fn> <error>` — add error return through callers\n\n")

	// Project-specific section
	b.WriteString("## This Project\n\n")
	b.WriteString(fmt.Sprintf("- **Language:** %s\n", project.Language))
	b.WriteString(fmt.Sprintf("- **Packages:** %d (see manifest.aid for full list)\n", project.PackageCount))

	// Detect entry points
	entryPoints := detectEntryPoints(project)
	if len(entryPoints) > 0 {
		b.WriteString(fmt.Sprintf("- **Entry points:** %s\n", strings.Join(entryPoints, ", ")))
	}

	if stats != nil {
		b.WriteString(fmt.Sprintf("- **Graph:** %d nodes, %d call edges\n", stats.NodeCount, stats.CallEdges))
	}
	b.WriteString("\n")

	// Field reference
	b.WriteString("## AID Field Quick Reference\n\n")
	b.WriteString("| Field | Meaning |\n")
	b.WriteString("|-------|--------|\n")
	b.WriteString("| @fn / @sig | Function signature |\n")
	b.WriteString("| @calls | Functions this calls (for tracing call chains) |\n")
	b.WriteString("| @type / @fields | Struct with its fields |\n")
	b.WriteString("| @trait / @requires | Interface contract |\n")
	b.WriteString("| @workflow / @steps | Multi-step data flow |\n")
	b.WriteString("| @invariants | Constraints that always hold |\n")
	b.WriteString("| @antipatterns | Common mistakes to avoid |\n")
	b.WriteString("| @error_map | Error taxonomy with classification |\n")
	b.WriteString("| @lock | Mutex documentation with ordering |\n")
	b.WriteString("| @pre / @post | Preconditions and postconditions |\n")

	path := filepath.Join(project.AidDir, "AGENTS.md")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// InstallSkills detects which AI tools are configured and installs the appropriate skill files.
func InstallSkills(project *detect.Project) []string {
	var installed []string

	// Claude Code — install skill if .claude/ exists
	claudeDir := filepath.Join(project.SourceRoot, ".claude", "skills")
	if _, err := os.Stat(filepath.Dir(claudeDir)); err == nil {
		if err := os.MkdirAll(claudeDir, 0755); err == nil {
			skillPath := filepath.Join(claudeDir, "squire.md")
			if err := os.WriteFile(skillPath, []byte(claudeSkill), 0644); err == nil {
				installed = append(installed, ".claude/skills/squire.md")
			}
		}
	}

	// Cursor — install .cursorrules if Cursor config exists or user has .cursorrules already
	cursorRules := filepath.Join(project.SourceRoot, ".cursorrules")
	cursorDir := filepath.Join(project.SourceRoot, ".cursor")
	if _, err := os.Stat(cursorRules); err == nil {
		// .cursorrules exists — append squire section if not already present
		existing, _ := os.ReadFile(cursorRules)
		if !strings.Contains(string(existing), "squire query") {
			f, err := os.OpenFile(cursorRules, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\n\n" + cursorSection)
				f.Close()
				installed = append(installed, ".cursorrules (appended)")
			}
		}
	} else if _, err := os.Stat(cursorDir); err == nil {
		// .cursor/ exists but no .cursorrules — create one
		if err := os.WriteFile(cursorRules, []byte(cursorSection), 0644); err == nil {
			installed = append(installed, ".cursorrules (created)")
		}
	}

	// GitHub Copilot — install if .github/ exists
	ghDir := filepath.Join(project.SourceRoot, ".github")
	if _, err := os.Stat(ghDir); err == nil {
		copilotPath := filepath.Join(ghDir, "copilot-instructions.md")
		if _, err := os.Stat(copilotPath); err != nil {
			// Doesn't exist yet — create it
			if err := os.WriteFile(copilotPath, []byte(copilotInstructions), 0644); err == nil {
				installed = append(installed, ".github/copilot-instructions.md")
			}
		}
	}

	return installed
}

const claudeSkill = `---
name: squire
description: Use squire to query the semantic code graph, understand symbol relationships, perform precise refactoring, and estimate implementation effort. Squire embeds AID documentation, Cartograph graph queries, and Chisel refactoring in a single tool.
trigger: when you need to understand code relationships, trace call chains, find what depends on a symbol, rename/move/propagate changes across a codebase, estimate story points or effort for a plan, or when .aidocs/ directory exists in the project
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation (AID files), a semantic graph query engine (Cartograph), precise refactoring (Chisel), and effort estimation in a single binary.

## When to Use Squire

- **Before reading source code**: Check if .aidocs/ exists. If it does, read the .aid file for the relevant package FIRST.
- **Tracing dependencies**: Use squire query instead of grepping.
- **Refactoring**: Use squire refactor instead of manual find-and-replace.
- **Estimating effort**: Use squire estimate to analyze a plan and get a story point size.

## Read AID Documentation

` + "`" + `` + "`" + `` + "`" + `bash
cat .aidocs/manifest.aid          # package index
cat .aidocs/<package>.aid         # package documentation
` + "`" + `` + "`" + `` + "`" + `

AID files contain: @fn/@sig (signatures), @calls (call graph), @type/@fields (structs), @trait (interfaces), @workflow (data flows), @invariants, @antipatterns, @error_map, @lock.

## Query the Semantic Graph

` + "`" + `` + "`" + `` + "`" + `bash
squire query callstack <function> --up     # who calls this?
squire query callstack <function> --down   # what does this call?
squire query depends <Type>                # what depends on this type?
squire query search "<pattern>"            # find by name (glob/regex)
squire query list <module>                 # list everything in a module
squire query field <Type.Field>            # what touches this field?
squire query errors <ErrorType>            # what produces this error?
squire query stats                         # graph statistics
` + "`" + `` + "`" + `` + "`" + `

## Refactor (dry-run by default)

` + "`" + `` + "`" + `` + "`" + `bash
squire refactor rename <old> <new>                    # preview rename
squire refactor rename <old> <new> --apply            # apply rename
squire refactor move <symbol> <package>               # move symbol
squire refactor propagate <function> <error-type>     # add error returns
` + "`" + `` + "`" + `` + "`" + `

## Estimate Effort

When the user asks you to estimate effort, story points, or complexity for a plan or feature, use squire estimate. It analyzes the semantic graph to count affected files, functions, packages, and cross-cutting concerns, then produces a T-shirt size (TINY/SMALL/MEDIUM/LARGE/XLARGE).

` + "`" + `` + "`" + `` + "`" + `bash
# From a plan file:
squire estimate --plan plan.md

# From explicit symbols mentioned in the plan:
squire estimate SnapshotInfo IsVisibleToSnapshot SetSnapshot

# Machine-readable output:
squire estimate --plan plan.md --format json
` + "`" + `` + "`" + `` + "`" + `

When asked to estimate, you can also write the plan to a temp file and pass it:
1. Write the plan text to a temp file
2. Run squire estimate --plan <tempfile>
3. Report the result to the user

## Strategy

1. Read .aidocs/manifest.aid to identify relevant packages
2. Read the .aid file for each package before source code
3. Use squire query to trace dependencies
4. Read source only for implementation details not in AID
5. After changes, run squire generate to update .aidocs/
`

const cursorSection = `# Squire — Structured Code Documentation

This project uses .aidocs/ for AI-readable code documentation. Read .aidocs/manifest.aid for the package index. Read a package's .aid file before its source code.

Use squire query for dependency tracing:
- squire query callstack <function> --up (callers)
- squire query callstack <function> --down (callees)
- squire query depends <Type> (dependents)
- squire query search "<pattern>" (find by name)

Use squire refactor for precise changes (dry-run by default, --apply to modify):
- squire refactor rename <old> <new>
- squire refactor move <symbol> <package>
- squire refactor propagate <fn> <error>

Use squire estimate to size implementation plans:
- squire estimate --plan plan.md (from a plan file)
- squire estimate Symbol1 Symbol2 (from explicit symbols)

Run squire generate after code changes to update .aidocs/.
`

const copilotInstructions = `# Squire — Structured Code Documentation

This project uses Squire for AI-readable code documentation in .aidocs/.

## How to Use

1. Read .aidocs/manifest.aid for the package index
2. Read a package's .aid file before reading its source code
3. AID files contain: function signatures (@fn), call graphs (@calls), type definitions (@type), workflows (@workflow), invariants (@invariants), antipatterns (@antipatterns), error taxonomy (@error_map), lock ordering (@lock)

## Commands (if terminal available)

- squire query callstack <function> --up — find callers
- squire query depends <Type> — find dependents
- squire query search "<pattern>" — find by name
- squire refactor rename <old> <new> — preview rename
- squire estimate --plan plan.md — estimate effort for a plan
- squire generate — update .aidocs/ after changes
`

func detectEntryPoints(project *detect.Project) []string {
	var entries []string
	for _, pkg := range project.Packages {
		// Look for main packages (common Go pattern: cmd/*)
		if pkg.Name == "main" || strings.HasPrefix(pkg.Dir, "cmd/") || strings.HasPrefix(pkg.Dir, "cmd"+string(os.PathSeparator)) {
			entries = append(entries, pkg.Dir)
		}
	}
	return entries
}
