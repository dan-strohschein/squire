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
	b.WriteString("3. **Read targeted source**: Use `squire show <symbol>` to extract just the function you need\n")
	b.WriteString("   instead of reading entire files. Use `squire excerpt file.go:Func1,Func2` for file-centric extraction.\n")
	b.WriteString("4. **Query the graph**: Use `squire query` to trace dependencies without reading source:\n")
	b.WriteString("   - `squire query callstack <function> --up` — who calls this?\n")
	b.WriteString("   - `squire query callstack <function> --down` — what does this call?\n")
	b.WriteString("   - `squire query depends <Type>` — what depends on this type?\n")
	b.WriteString("   - `squire query field <Type.Field>` — what reads or writes this field?\n")
	b.WriteString("   - `squire query errors <ErrorType>` — what produces this error?\n")
	b.WriteString("   - `squire query effects <function>` — what side effects does this have?\n")
	b.WriteString("   - `squire query search <pattern>` — find by name\n")
	b.WriteString("5. **Analyze changes**: Use `squire impact` to check blast radius before pushing.\n")
	b.WriteString("   Use `squire estimate <Symbol>` to size implementation work.\n")
	b.WriteString("   Use `squire stale` to check which AID claims have outdated source references.\n")
	b.WriteString("6. **Refactor with precision**: Use `squire refactor` for codebase-wide changes:\n")
	b.WriteString("   - `squire refactor rename <old> <new>` — rename across all files\n")
	b.WriteString("   - `squire refactor move <symbol> <dest>` — move between packages\n")
	b.WriteString("   - `squire refactor propagate <fn> <error>` — add error return through callers\n")
	b.WriteString("   - `squire refactor extract <fn> <package>` — extract function and deps to new package\n\n")

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

	// Claude Code — install skills if .claude/ exists
	claudeDir := filepath.Join(project.SourceRoot, ".claude", "skills")
	if _, err := os.Stat(filepath.Dir(claudeDir)); err == nil {
		if err := os.MkdirAll(claudeDir, 0755); err == nil {
			skillPath := filepath.Join(claudeDir, "squire.md")
			if err := os.WriteFile(skillPath, []byte(claudeSkill), 0644); err == nil {
				installed = append(installed, ".claude/skills/squire.md")
			}
			sessionPath := filepath.Join(claudeDir, "squire-session.md")
			if err := os.WriteFile(sessionPath, []byte(sessionSkill), 0644); err == nil {
				installed = append(installed, ".claude/skills/squire-session.md")
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
description: Use squire to read source code efficiently, query the semantic graph, analyze impact, perform refactoring, and estimate effort. Squire embeds AID, Cartograph, and Chisel in a single tool.
trigger: when you need to read a function's source code, understand code relationships, trace call chains, analyze blast radius of changes, find what depends on a symbol, rename/move/propagate/extract, estimate effort for a plan, check AID staleness, or when .aidocs/ directory exists
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation, a semantic graph engine, source code extraction, impact analysis, refactoring, and effort estimation in a single binary.

## When to Use Squire

- **Reading source code**: Use squire show instead of reading entire files — it extracts just the function you need.
- **Before reading source code**: Check if .aidocs/ exists. Read the .aid file for the relevant package FIRST.
- **Tracing dependencies**: Use squire query instead of grepping.
- **Before pushing changes**: Use squire impact to find dependencies you might have missed.
- **Checking staleness**: Use squire stale to see which AID claims have outdated source references.
- **Refactoring**: Use squire refactor instead of manual find-and-replace.
- **Estimating effort**: Use squire estimate to analyze a plan and get a story point size.

## Show Source Code (PREFER THIS OVER READING FILES)

Instead of reading an entire 500-line file to find one function, use squire show:

` + "`" + `` + "`" + `` + "`" + `bash
squire show HandleRequest              # show just this function's source code
squire show Resolver.Resolve           # show a method
squire show EditSet                    # show a type definition
squire show HandleRequest NewHandler   # show multiple symbols
` + "`" + `` + "`" + `` + "`" + `

This returns only the function body (10-50 lines) instead of the entire file. Use this FIRST before falling back to reading full files. It saves 80-90% of read tokens.

## Excerpt Source Code (FILE-CENTRIC EXTRACTION)

When you know the file and want specific functions from it, use squire excerpt:

` + "`" + `` + "`" + `` + "`" + `bash
squire excerpt service.go:Create,retryOne   # two functions from one file
squire excerpt service.go                   # all declarations in file
squire excerpt svc.go:Create job.go:Run     # across multiple files
` + "`" + `` + "`" + `` + "`" + `

Use excerpt when you know the file path. Use show when you know the symbol name.

## Digest Session Findings (COMPRESS CONTEXT)

When investigating a codebase, write findings to a scratch file, then compress:

` + "`" + `` + "`" + `` + "`" + `bash
squire digest --from findings.md --task "Bug hunt"
squire digest --from findings.md --out digest.md
` + "`" + `` + "`" + `` + "`" + `

Digest cross-references findings against the AID graph, adding callers/callees and package context. Use before continuation sessions.

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
squire query field <Type.Field>            # what touches this field?
squire query errors <ErrorType>            # what produces this error?
squire query effects <function>            # what side effects?
squire query search "<pattern>"            # find by name (glob/regex)
squire query list <module>                 # list everything in a module
squire query stats                         # graph statistics
` + "`" + `` + "`" + `` + "`" + `

## Impact Analysis (before pushing)

Check what depends on your changes that you might have missed:

` + "`" + `` + "`" + `` + "`" + `bash
squire impact                  # analyze uncommitted git changes
squire impact --staged         # analyze staged changes only
squire impact SnapshotInfo     # what breaks if I change this type?
` + "`" + `` + "`" + `` + "`" + `

## Staleness (detailed claim-level checking)

` + "`" + `` + "`" + `` + "`" + `bash
squire stale                   # check which [src:] references are outdated
` + "`" + `` + "`" + `` + "`" + `

Use squire stale for detailed per-claim staleness. Use squire status for package-level overview.

## Refactor (dry-run by default)

` + "`" + `` + "`" + `` + "`" + `bash
squire refactor rename <old> <new>                    # preview rename
squire refactor rename <old> <new> --apply            # apply rename
squire refactor move <symbol> <package>               # move symbol
squire refactor propagate <function> <error-type>     # add error returns
squire refactor extract <function> <new-package>      # extract to new package
` + "`" + `` + "`" + `` + "`" + `

For type-aware refactoring on generic symbol names, add --lsp-cmd:

` + "`" + `` + "`" + `` + "`" + `bash
squire refactor rename Get Fetch --lsp-cmd "gopls serve"      # Go
squire refactor rename Get Fetch --lsp-cmd "pyright"           # Python
squire refactor rename Get Fetch --lsp-cmd "rust-analyzer"     # Rust
` + "`" + `` + "`" + `` + "`" + `

## Estimate Effort

When the user asks you to estimate effort, story points, or complexity for a plan or feature, follow this two-step process:

### Step 1: Validate the plan for specificity BEFORE running squire

Review the plan text and check: does it name specific types, functions, or interfaces that will change? If not, the plan is too vague to estimate. Ask clarifying questions first.

**Signs of a vague plan (DO NOT submit to squire yet):**
- Uses only general verbs: "refactor", "clean up", "improve", "fix"
- Refers to systems by domain name only: "the notification system", "the auth module"
- No specific type names, function names, or interface names mentioned

**When the plan is vague, ask questions like:**
- "What specifically should change? Which types or interfaces?"
- "Are you adding new fields, changing signatures, or moving code?"
- "What's the end state?"

Use ` + "`" + `squire query search "<keyword>"` + "`" + ` to help the user find relevant symbols.

### Step 2: Run squire estimate once the plan names specific symbols

` + "`" + `` + "`" + `` + "`" + `bash
squire estimate --plan /tmp/plan.md
squire estimate NotificationServiceImpl DLQRetryJob Handler
` + "`" + `` + "`" + `` + "`" + `

If squire returns UNCLEAR, relay that to the user and go back to Step 1.

## Strategy

1. Read .aidocs/manifest.aid to identify relevant packages
2. Read the .aid file for those packages before source code
3. Use squire query to trace dependencies and call chains
4. Use squire show or squire excerpt to read specific functions — NOT entire files
5. Only read full files when you need broad context (rare)
6. Write findings to a scratch file as you go — compress with squire digest before continuing
7. Use squire impact before pushing to check for missed dependencies
8. Use squire stale to check if AID claims are still accurate
9. After changes, run squire generate to update .aidocs/
`

const sessionSkill = `---
name: session-management
description: Teaches checkpoint discipline for multi-session analysis. Write findings incrementally, use squire excerpt/show instead of full file reads, signal DONE or CHECKPOINT when appropriate.
trigger: when squire-session is orchestrating, or when investigating a large codebase and accumulating many tool results
---

# Session Management — Checkpoint Discipline

You are running inside a multi-session analysis workflow. Each session has a limited number of turns. Your findings are compressed between sessions via squire digest, so the next session starts with minimal context.

## Rules

### 1. Write findings incrementally
Write observations to the designated findings file as you discover them. Use this format:

` + "```" + `markdown
## file.go:line - SymbolName
Your observation here. Be specific about what you found.
` + "```" + `

Do NOT wait until the end to write everything at once. Write after each significant discovery.

### 2. Use squire tools to minimize context
- squire excerpt file.go:Func1,Func2 — extract specific functions from a file (PREFER THIS)
- squire show SymbolName — extract a function by name from the graph
- squire query callstack FuncName --up — trace callers without reading source
- squire query depends TypeName — find dependents without grep
- Do NOT read entire files unless you need broad context that squire cannot provide

### 3. Check for prior digest
If the prompt includes "Prior session digest:", read it carefully. It contains compressed findings from earlier sessions. Do NOT re-read files or re-investigate issues already covered in the digest.

### 4. Signal completion
- Write ## DONE at the end of the findings file when the task is complete
- Write ## CHECKPOINT when you have accumulated enough findings for a useful digest but the task is not yet complete

### 5. Open questions
Prefix unresolved questions with ? so they are extracted into the digest.
`

const cursorSection = `# Squire — AI Code Assistant Toolkit

This project uses .aidocs/ for AI-readable code documentation. Read .aidocs/manifest.aid for the package index. Read a package's .aid file before its source code.

Use squire show to read specific functions (not entire files):
- squire show HandleRequest (extracts just that function's source)
- squire show Type.Method (show a method)
- squire excerpt file.go:Func1,Func2 (extract from a file)

Use squire query for dependency tracing:
- squire query callstack <function> --up (callers)
- squire query callstack <function> --down (callees)
- squire query depends <Type> (dependents)
- squire query field <Type.Field> (field readers/writers)
- squire query errors <ErrorType> (error producers)
- squire query effects <function> (side effects)
- squire query search "<pattern>" (find by name)

Use squire impact to check blast radius before pushing:
- squire impact (analyze uncommitted changes)
- squire impact SymbolName (what breaks if I change this?)

Use squire refactor for precise changes (dry-run by default, --apply to modify):
- squire refactor rename <old> <new>
- squire refactor move <symbol> <package>
- squire refactor propagate <fn> <error>
- squire refactor extract <fn> <new-package>

Use squire estimate to size implementation plans:
- squire estimate --plan plan.md (from a plan file)
- squire estimate Symbol1 Symbol2 (from explicit symbols)

Use squire stale to check which AID claims have outdated source references.
Run squire generate after code changes to update .aidocs/.
`

const copilotInstructions = `# Squire — AI Code Assistant Toolkit

This project uses Squire for AI-readable code documentation in .aidocs/.

## How to Use

1. Read .aidocs/manifest.aid for the package index
2. Read a package's .aid file before reading its source code
3. Use squire show <symbol> to read just one function instead of an entire file

## Commands (if terminal available)

### Reading source
- squire show <symbol> — read just that function's source (not the whole file)
- squire excerpt file.go:Func1,Func2 — extract specific functions from a file

### Querying the graph
- squire query callstack <function> --up — find callers
- squire query callstack <function> --down — find callees
- squire query depends <Type> — find dependents
- squire query field <Type.Field> — find field readers/writers
- squire query errors <ErrorType> — find error producers
- squire query effects <function> — find side effects
- squire query search "<pattern>" — find by name

### Analysis
- squire impact — check what depends on your changes
- squire impact <Symbol> — what breaks if this changes
- squire estimate --plan plan.md — estimate effort for a plan
- squire stale — check which AID claims are outdated

### Refactoring (dry-run by default, --apply to modify)
- squire refactor rename <old> <new> — preview rename
- squire refactor move <symbol> <package> — move between packages
- squire refactor propagate <fn> <error> — add error returns
- squire refactor extract <fn> <package> — extract to new package

### Maintenance
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
