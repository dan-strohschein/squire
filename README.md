# Squire

AI code assistant toolkit. One binary that generates structured documentation for your codebase, provides a semantic query engine, and enables precise refactoring — so your AI assistant spends less time exploring and more time helping.

**Squire embeds three tools:**
- [AID](https://github.com/dan-strohschein/AID-Docs) — structured documentation format designed for AI agents
- [Cartograph](https://github.com/dan-strohschein/Cartograph) — semantic graph engine for tracing dependencies and call chains
- [Chisel](https://github.com/dan-strohschein/chisel) — codebase-wide refactoring powered by the semantic graph

## Why

AI coding assistants waste 12-19% of their token budget just figuring out where things are in your codebase. Squire generates a structured index that lets them skip the exploration and go straight to the answer.

[Benchmark results](https://github.com/dan-strohschein/chisel/blob/main/benchmark-comparison.md): 19% fewer tokens on private codebases, 12% on open-source projects the model already knows.

## Install

```bash
# Option 1: Shell script (downloads single binary)
curl -sSL https://raw.githubusercontent.com/dan-strohschein/squire/main/install.sh | sh

# Option 2: Go install
go install github.com/dan-strohschein/squire@latest
```

For Go projects, you also need the Go generator:
```bash
go install github.com/dan-strohschein/aid-gen-go@latest
```

## Quick Start

```bash
cd /path/to/your/project
squire init
```

That's it. Squire detects your project language, generates `.aidocs/` with structured documentation, and creates an `AGENTS.md` file that teaches any AI assistant how to use it.

```
Detecting project...
  Language:    Go (found go.mod)
  Module:      github.com/example/myservice
  Packages:    42 packages

Generating AID files...
  ✓ 42 packages → 42 .aid files
  ✓ manifest.aid generated
  ✓ Validation passed

Building semantic graph...
  ✓ 1,247 nodes | 3,891 edges | 2,104 call edges

Configuring AI integration...
  ✓ .aidocs/AGENTS.md created

Done! Your AI assistant can now use .aidocs/ for structured codebase knowledge.
```

## Commands

### For developers

| Command | What it does |
|---------|-------------|
| `squire init` | Set up AID for this project (detect language, generate, configure) |
| `squire generate` | Regenerate `.aidocs/` from current source (incremental — only updates changed packages) |
| `squire status` | Show what's generated, what's stale, what's missing |
| `squire doctor` | Verify installation and project health |
| `squire install <tool>` | Install a language-specific generator |
| `squire upgrade` | Update all installed tools to latest |

### For AI agents

| Command | What it does |
|---------|-------------|
| `squire query callstack <fn> --up` | Who calls this function? |
| `squire query callstack <fn> --down` | What does this function call? |
| `squire query depends <Type>` | What depends on this type? |
| `squire query search <pattern>` | Find functions/types by name |
| `squire query list <module>` | List everything in a module |
| `squire refactor rename <old> <new>` | Rename a symbol across the codebase |
| `squire refactor move <sym> <dest>` | Move a symbol to another package |
| `squire refactor propagate <fn> <err>` | Add error return through callers |

All refactor commands default to **dry-run** (preview the diff). Pass `--apply` to modify files.

## What gets generated

```
.aidocs/
├── manifest.aid          # Package index — start here
├── <package>.aid         # One per package: signatures, call graphs, types
└── AGENTS.md             # Instructions for any AI assistant
```

Each `.aid` file contains:
- **Function signatures** (`@fn`, `@sig`) — what functions exist and what they accept/return
- **Call graphs** (`@calls`) — who calls what, enabling `squire query callstack`
- **Type definitions** (`@type`, `@fields`) — structs and their fields
- **Interfaces** (`@trait`, `@requires`) — contracts that types implement

With [L2 enrichment](https://github.com/dan-strohschein/AID-Docs) (AI-assisted, optional), files also contain:
- **Workflows** (`@workflow`) — multi-step data flows
- **Invariants** (`@invariants`) — constraints that always hold
- **Antipatterns** (`@antipatterns`) — common mistakes to avoid
- **Error taxonomy** (`@error_map`) — error classification and caller behavior
- **Lock documentation** (`@lock`) — mutex ordering and deadlock avoidance

## Supported Languages

| Language | Generator | Status |
|----------|-----------|--------|
| Go | `aid-gen-go` | Stable |
| TypeScript | `aid-gen-ts` | Stable |
| C# | `aid-gen-cs` | Stable |
| Python | `aid-gen` | Stable |
| Rust | — | Planned |
| Java/Kotlin | — | Planned |

Squire auto-detects the language from project markers (`go.mod`, `package.json`, `*.csproj`, `pyproject.toml`). Generators are installed on demand.

## How AI agents use it

When you run `squire init`, it creates `.aidocs/AGENTS.md` — a plain markdown file that any AI assistant reads as project context. It tells the agent:

1. Read `.aidocs/manifest.aid` first for the package map
2. Read a package's `.aid` file before reading its source code
3. Use `squire query` to trace dependencies without reading source
4. Use `squire refactor` for precise codebase-wide changes

This works with **any AI coding tool** that reads project files: Claude Code, Cursor, GitHub Copilot, Windsurf, or any future tool.

## Incremental updates

After code changes, run:

```bash
squire generate
```

Squire compares each `.aid` file's `@code_version` against the current git HEAD for that package. Only stale packages are regenerated.

```
Scanning for changes since last generation...
  Modified: 3 packages (bundle, server, planner)
  Unchanged: 39 packages

Regenerating...
  ✓ 3 files updated in .aidocs/
```

## Examples

### Query: trace callers of a function
```bash
$ squire query callstack FindSourceLocations --up

FindSourceLocations
   └─ Resolver.locateAll [Calls]
      └─ Resolver.resolveRename [Calls]
         └─ Resolver.Resolve [Calls]
```

### Query: find by name
```bash
$ squire query search "Handle*"

  Function (12):
    HandleConnection — Manages a single TCP client connection
    HandleCommand — Routes a parsed command to the correct handler
    ...
```

### Refactor: rename across codebase (dry-run)
```bash
$ squire refactor rename EditKind RefactorEditKind

Modified 2 file(s), 5 edit(s) applied, 1 AID file(s) updated (dry-run)

--- a/edit/types.go
+++ b/edit/types.go
-type EditKind int
+type RefactorEditKind int

Dry run — no files modified. Pass --apply to make changes.
```

### Check project health
```bash
$ squire doctor

Checking squire installation...
  ✓ squire v1.0.0
  ✓ cartograph engine embedded
  ✓ aid-gen-go (/usr/local/bin/aid-gen-go)

Checking project (github.com/example/myservice — Go)...
  ✓ .aidocs/ exists (42 .aid files)
  ✓ manifest.aid present
  ✓ AGENTS.md present
  ✓ Semantic graph loads (1,247 nodes, 2,104 call edges)
  ✓ Coverage: 42/42 packages have AID files

All checks passed. Your project is ready for AI-assisted development.
```

## Architecture

Squire is a single Go binary (~4MB) that embeds:
- The [aidkit](https://github.com/dan-strohschein/aidkit) parser for validation and manifest generation
- The [Cartograph](https://github.com/dan-strohschein/Cartograph) graph engine for semantic queries
- The [Chisel](https://github.com/dan-strohschein/chisel) refactoring engine for precise code changes

Language-specific generators (`aid-gen-go`, `aid-gen-ts`, etc.) are separate binaries managed by `squire install`.

## License

MIT
