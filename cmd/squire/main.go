package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/chisel/edit"
	"github.com/dan-strohschein/chisel/patch"
	"github.com/dan-strohschein/chisel/resolve"
	"github.com/dan-strohschein/squire/internal/agents"
	"github.com/dan-strohschein/squire/internal/carto"
	"github.com/dan-strohschein/squire/internal/detect"
	"github.com/dan-strohschein/squire/internal/generate"
	"github.com/dan-strohschein/squire/internal/refactor"
	"github.com/dan-strohschein/squire/internal/status"
	"github.com/dan-strohschein/squire/internal/tools"
	"github.com/dan-strohschein/squire/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "init":
		cmdInit(args)
	case "generate":
		cmdGenerate(args)
	case "status":
		cmdStatus(args)
	case "doctor":
		cmdDoctor(args)
	case "query":
		cmdQuery(args)
	case "refactor":
		cmdRefactor(args)
	case "install":
		cmdInstall(args)
	case "upgrade":
		cmdUpgrade()
	case "version":
		cmdVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func cmdInit(args []string) {
	dir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = args[0]
	}

	// Step 1: Detect project
	fmt.Println("Detecting project...")
	project, err := detect.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Language:    %s (%s)\n", project.Language, project.Evidence)
	fmt.Printf("  Module:      %s\n", project.Module)
	fmt.Printf("  Packages:    %d packages in %s\n", project.PackageCount, project.SourceRoot)
	fmt.Println()

	// Step 2: Check generator is available
	genLang := strings.ToLower(project.Language)
	_, genErr := generate.FindGenerator(genLang)
	if genErr != nil {
		fmt.Printf("  Generator for %s not found.\n", project.Language)
		fmt.Printf("  Install with: squire install aid-gen-%s\n", genLang)
		fmt.Printf("  Or:           go install github.com/dan-strohschein/aid-gen-%s@latest\n", genLang)
		os.Exit(1)
	}

	// Step 3: Generate L1 AID
	fmt.Println("Generating AID files...")
	result, err := generate.Generate(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating AID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ %d packages → %d .aid files in .aidocs/\n", result.PackagesProcessed, result.FilesGenerated)
	fmt.Printf("  ✓ manifest.aid generated\n")

	// Step 3: Validate
	warnings := result.Warnings
	if len(warnings) > 0 {
		fmt.Printf("  ⚠ %d warnings\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("    %s\n", w)
		}
	} else {
		fmt.Printf("  ✓ Validation passed\n")
	}
	fmt.Println()

	// Step 4: Build graph stats
	fmt.Println("Building semantic graph...")
	graphStats, err := generate.LoadGraphStats(project.AidDir)
	if err == nil {
		fmt.Printf("  ✓ %d nodes | %d edges | %d call edges\n",
			graphStats.NodeCount, graphStats.EdgeCount, graphStats.CallEdges)
	}
	fmt.Println()

	// Step 5: Generate AGENTS.md
	fmt.Println("Configuring AI integration...")
	err = agents.Generate(project, graphStats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Could not generate AGENTS.md: %v\n", err)
	} else {
		fmt.Printf("  ✓ .aidocs/AGENTS.md created\n")
	}
	fmt.Println()

	fmt.Printf("Done! Your AI assistant can now use .aidocs/ for structured codebase knowledge.\n")
	fmt.Printf("Run `squire generate` after significant code changes to keep .aidocs/ current.\n")
}

func cmdGenerate(args []string) {
	dir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = args[0]
	}

	project, err := detect.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check staleness
	stale, fresh, missing, err := status.CheckStaleness(project)
	if err != nil {
		// First run or no existing .aidocs — regenerate everything
		fmt.Println("Generating all AID files...")
	} else {
		fmt.Printf("Scanning for changes since last generation...\n")
		fmt.Printf("  Modified: %d packages\n", len(stale))
		fmt.Printf("  New: %d packages\n", len(missing))
		fmt.Printf("  Unchanged: %d packages\n", len(fresh))
		fmt.Println()

		if len(stale) == 0 && len(missing) == 0 {
			fmt.Println("All .aidocs/ files are up to date.")
			return
		}
		fmt.Println("Regenerating...")
	}

	result, err := generate.Generate(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  ✓ %d files updated in .aidocs/\n", result.FilesGenerated)

	// Rebuild AGENTS.md
	graphStats, _ := generate.LoadGraphStats(project.AidDir)
	_ = agents.Generate(project, graphStats)
}

func cmdStatus(args []string) {
	dir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = args[0]
	}

	project, err := detect.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if .aidocs/ even exists
	if _, statErr := os.Stat(project.AidDir); statErr != nil {
		fmt.Printf(".aidocs/ not found for %s (%s)\n", project.Module, project.Language)
		fmt.Printf("\nRun `squire init` to set up AID documentation.\n")
		return
	}

	stale, fresh, missing, err := status.CheckStaleness(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking status: %v\n", err)
		os.Exit(1)
	}

	// Count .aid files directly
	aidCount := 0
	l2Count := 0
	aids, _ := os.ReadDir(project.AidDir)
	for _, f := range aids {
		if strings.HasSuffix(f.Name(), ".aid") && f.Name() != "manifest.aid" {
			aidCount++
		}
	}

	total := len(stale) + len(fresh) + len(missing)
	fmt.Printf(".aidocs/ status for %s (%s)\n\n", project.Module, project.Language)
	fmt.Printf("  Packages detected: %d\n", project.PackageCount)
	fmt.Printf("  AID files:         %d\n", aidCount)

	if total > 0 {
		fmt.Printf("  Up to date:        %d/%d", len(fresh), total)
		if len(fresh) == total {
			fmt.Printf("  ✓")
		}
		fmt.Println()
	}
	if len(stale) > 0 {
		fmt.Printf("  Stale:             %d  (%s)\n", len(stale), strings.Join(stale, ", "))
	}
	if len(missing) > 0 {
		fmt.Printf("  Missing:           %d  (%s)\n", len(missing), strings.Join(missing, ", "))
	}
	if l2Count > 0 {
		fmt.Printf("  L2 enriched:       %d/%d\n", l2Count, aidCount)
	}
	fmt.Println()

	graphStats, err := generate.LoadGraphStats(project.AidDir)
	if err == nil {
		fmt.Printf("  Graph:     %d nodes, %d edges, %d call edges\n",
			graphStats.NodeCount, graphStats.EdgeCount, graphStats.CallEdges)
	}

	agentsPath := project.AidDir + "/AGENTS.md"
	if _, statErr := os.Stat(agentsPath); statErr == nil {
		fmt.Printf("  AGENTS.md: ✓ present\n")
	} else {
		fmt.Printf("  AGENTS.md: ✗ missing (run `squire init`)\n")
	}

	if len(stale) > 0 || len(missing) > 0 {
		fmt.Printf("\nRun `squire generate` to update stale/missing packages.\n")
	} else {
		fmt.Printf("\nAll packages up to date.\n")
	}
}

func cmdDoctor(args []string) {
	problems := 0
	warnings := 0

	fmt.Println("Checking squire installation...")
	fmt.Printf("  ✓ squire %s\n", version.Version)
	fmt.Printf("  ✓ cartograph engine embedded\n")

	// Check for aid-gen-go
	genPath, err := generate.FindGenerator("go")
	if err == nil {
		fmt.Printf("  ✓ aid-gen-go (%s)\n", genPath)
	} else {
		fmt.Printf("  ✗ aid-gen-go not found\n")
		fmt.Printf("    Install: go install github.com/dan-strohschein/aid-gen-go@latest\n")
		problems++
	}

	fmt.Println()

	// Check project
	dir := "."
	project, err := detect.Detect(dir)
	if err != nil {
		fmt.Printf("Checking project...\n")
		fmt.Printf("  ✗ No project detected in current directory\n")
		fmt.Printf("    squire works in directories with go.mod, package.json, pyproject.toml, or .csproj\n")
		problems++
		printDoctorSummary(problems, warnings)
		return
	}

	fmt.Printf("Checking project (%s — %s)...\n", project.Module, project.Language)

	if _, err := os.Stat(project.AidDir); err == nil {
		aidCount := 0
		aids, _ := os.ReadDir(project.AidDir)
		for _, f := range aids {
			if strings.HasSuffix(f.Name(), ".aid") {
				aidCount++
			}
		}
		fmt.Printf("  ✓ .aidocs/ exists (%d .aid files)\n", aidCount)
	} else {
		fmt.Printf("  ✗ .aidocs/ not found\n")
		fmt.Printf("    Run `squire init` to generate AID documentation\n")
		problems++
		printDoctorSummary(problems, warnings)
		return
	}

	if _, err := os.Stat(project.AidDir + "/manifest.aid"); err == nil {
		fmt.Printf("  ✓ manifest.aid present\n")
	} else {
		fmt.Printf("  ✗ manifest.aid missing\n")
		fmt.Printf("    Run `squire init` to regenerate\n")
		problems++
	}

	if _, err := os.Stat(project.AidDir + "/AGENTS.md"); err == nil {
		fmt.Printf("  ✓ AGENTS.md present\n")
	} else {
		fmt.Printf("  ✗ AGENTS.md missing\n")
		fmt.Printf("    Run `squire init` to regenerate\n")
		problems++
	}

	graphStats, err := generate.LoadGraphStats(project.AidDir)
	if err == nil {
		fmt.Printf("  ✓ Semantic graph loads (%d nodes, %d call edges)\n", graphStats.NodeCount, graphStats.CallEdges)
	} else {
		fmt.Printf("  ⚠ Graph failed to load: %v\n", err)
		warnings++
	}

	stale, fresh, missing, _ := status.CheckStaleness(project)
	generated := len(stale) + len(fresh)
	total := generated + len(missing)
	if total > 0 {
		fmt.Printf("  ✓ Coverage: %d/%d packages have AID files\n", generated, total)
	}
	if len(stale) > 0 {
		fmt.Printf("  ⚠ %d packages stale (%s)\n", len(stale), strings.Join(stale, ", "))
		fmt.Printf("    Run `squire generate` to update\n")
		warnings++
	}
	if len(missing) > 0 {
		fmt.Printf("  ⚠ %d packages missing AID files (%s)\n", len(missing), strings.Join(missing, ", "))
		fmt.Printf("    Run `squire generate` to add them\n")
		warnings++
	}

	printDoctorSummary(problems, warnings)
}

func printDoctorSummary(problems, warnings int) {
	fmt.Println()
	if problems > 0 {
		fmt.Printf("%d problem(s) found. Fix the issues above and run `squire doctor` again.\n", problems)
	} else if warnings > 0 {
		fmt.Printf("No problems. %d warning(s) — run `squire generate` to resolve.\n", warnings)
	} else {
		fmt.Println("All checks passed. Your project is ready for AI-assisted development.")
	}
}

func cmdQuery(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: squire query <command> [args]\n")
		fmt.Fprintf(os.Stderr, "Commands: callstack, depends, field, errors, effects, search, list, stats\n")
		os.Exit(1)
	}

	aidDir := detect.FindAidocs(".")
	if aidDir == "" {
		fmt.Fprintf(os.Stderr, "Error: no .aidocs/ directory found. Run `squire init` first.\n")
		os.Exit(1)
	}

	// Use embedded cartograph engine — no external binary needed
	if err := carto.Run(aidDir, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdRefactor(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: squire refactor <rename|move|propagate> <args> [--apply]\n")
		os.Exit(1)
	}

	subcmd := args[0]
	remaining := args[1:]

	// Parse flags
	apply := false
	format := "unified"
	includeComments := false
	var positional []string

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--apply":
			apply = true
		case "--format":
			if i+1 < len(remaining) {
				i++
				format = remaining[i]
			}
		case "--include-comments":
			includeComments = true
		default:
			positional = append(positional, remaining[i])
		}
	}

	// Find .aidocs/ and source dir
	aidDir := detect.FindAidocs(".")
	sourceDir := "."
	if aidDir != "" {
		sourceDir = filepath.Dir(aidDir)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: no .aidocs/ found. Falling back to grep-only resolution.\n")
	}

	// Build intent
	intent := resolve.Intent{
		AidDir:          aidDir,
		SourceDir:       sourceDir,
		IncludeComments: includeComments,
	}

	switch subcmd {
	case "rename":
		if len(positional) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: squire refactor rename <old> <new> [--apply]\n")
			os.Exit(1)
		}
		intent.Kind = resolve.Rename
		intent.Target = positional[0]
		intent.NewName = positional[1]

	case "move":
		if len(positional) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: squire refactor move <symbol> <destination> [--apply]\n")
			os.Exit(1)
		}
		intent.Kind = resolve.Move
		intent.Target = positional[0]
		intent.Destination = positional[1]

	case "propagate":
		if len(positional) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: squire refactor propagate <function> <error-type> [--apply]\n")
			os.Exit(1)
		}
		intent.Kind = resolve.Propagate
		intent.Target = positional[0]
		intent.ErrorType = positional[1]

	default:
		fmt.Fprintf(os.Stderr, "Unknown refactor command: %s\nAvailable: rename, move, propagate\n", subcmd)
		os.Exit(1)
	}

	// Phase 1: Resolve — use embedded cartograph
	querier := &refactor.EmbeddedGraphQuerier{}
	resolver := &resolve.Resolver{Graph: querier}

	resolution, err := resolver.Resolve(intent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving: %v\n", err)
		os.Exit(1)
	}

	// Phase 2: Generate edits (no LSP for now — use null resolver)
	var typeResolver resolve.TypeResolver = &resolve.NullResolver{}
	editSet, err := edit.GenerateEdits(resolution, typeResolver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating edits: %v\n", err)
		os.Exit(1)
	}

	// Phase 3: Apply or preview
	options := patch.PatchOptions{
		DryRun:       !apply,
		UpdateAid:    true,
		OutputFormat: format,
	}

	result, err := patch.Apply(editSet, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error applying: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(patch.FormatPatch(result, format))

	for _, w := range resolution.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if !apply {
		fmt.Fprintf(os.Stderr, "\nDry run — no files modified. Pass --apply to make changes.\n")
	}
}

func cmdInstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: squire install <tool>\n\n")
		fmt.Fprintf(os.Stderr, "Available tools:\n")
		for name := range tools.KnownTools {
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
		os.Exit(1)
	}

	if err := tools.Install(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdUpgrade() {
	fmt.Println("Checking for updates...")
	fmt.Println()

	fmt.Printf("  squire %s (upgrade squire itself with: go install github.com/dan-strohschein/squire@latest)\n", version.Version)
	fmt.Println()

	fmt.Println("Installed tools:")
	if err := tools.Upgrade(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdVersion() {
	fmt.Printf("squire %s\n", version.Version)
	fmt.Printf("  cartograph: embedded\n")
	fmt.Printf("  chisel: embedded\n")
	fmt.Printf("  aidkit parser: embedded\n")

	installed := tools.ListInstalled()
	if len(installed) > 0 {
		fmt.Printf("\nInstalled generators:\n")
		for name, ver := range installed {
			fmt.Printf("  %s %s\n", name, ver)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `squire — AI code assistant toolkit

Squire prepares your codebase for AI agents by generating structured
documentation (AID files), providing a semantic query engine, and
enabling precise codebase-wide refactoring.

Usage:
  squire init [dir]              Set up AID for this project (detect, generate, configure)
  squire generate [dir]          Regenerate .aidocs/ from current source (incremental)
  squire status [dir]            Show what's generated, stale, or missing
  squire doctor                  Verify installation and project health

  squire query <command> [args]  Query the semantic graph (embedded cartograph)
    callstack <fn> [--up|--down]   Trace callers or callees
    depends <Type>                 What depends on this type?
    search <pattern>               Find by name (glob/regex)
    list <module>                  List everything in a module
    stats                          Graph statistics

  squire refactor <command> [args]  Semantic refactoring (embedded chisel)
    rename <old> <new>              Rename a symbol across the codebase
    move <symbol> <dest>            Move a symbol to another package
    propagate <fn> <error>          Add error return through callers
    [--apply]                       Actually modify files (default: dry-run)

  squire install <tool>           Install a generator (aid-gen-go, aid-gen-ts, etc.)
  squire upgrade                 Update all installed tools to latest
  squire version                 Show version info

Examples:
  squire init                    Set up the current Go/TS/Python/C# project
  squire query callstack Serve --up
  squire refactor rename OldName NewName
`)
}
