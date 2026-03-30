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
