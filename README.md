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

### Setup & maintenance

| Command | What it does |
|---------|-------------|
| `squire init` | Set up AID for this project (detect language, generate, configure) |
| `squire generate` | Regenerate `.aidocs/` from current source (incremental — only updates changed packages) |
| `squire status` | Show what's generated, what's stale, what's missing |
| `squire stale` | Check AID claims against current source with detailed per-claim reporting |
| `squire doctor` | Verify installation, project health, and AID validation |
| `squire install <tool>` | Install a language-specific generator |
| `squire upgrade` | Update all installed tools to latest |

### Querying the semantic graph

| Command | What it does |
|---------|-------------|
| `squire query callstack <fn> --up` | Who calls this function? |
| `squire query callstack <fn> --down` | What does this function call? |
| `squire query depends <Type>` | What depends on this type? |
| `squire query field <Type.Field>` | Who reads or writes this field? |
| `squire query errors <ErrorType>` | Where is this error produced? |
| `squire query effects <fn>` | What side effects does this function have? |
| `squire query search <pattern>` | Find functions/types by name (glob/regex) |
| `squire query list <module>` | List everything in a module |
| `squire query stats` | Graph statistics |

### Reading source code

| Command | What it does |
|---------|-------------|
| `squire show <symbol>` | Show the source code of a function/type without reading the entire file |
| `squire excerpt <file>[:sym1,sym2]` | Extract specific symbol bodies from a file |

### Analysis

| Command | What it does |
|---------|-------------|
| `squire impact` | Blast radius of uncommitted changes — what you might have missed |
| `squire impact --staged` | Same, but for staged changes only |
| `squire impact <Symbol1> <Symbol2>` | What breaks if you change these symbols? |
| `squire estimate <Symbol1> <Symbol2>` | Estimate story points from graph analysis |
| `squire estimate --plan <file>` | Extract symbols from a plan file and estimate |
| `squire digest --from <findings.md>` | Compress session findings into AID-anchored summary |

### Refactoring

| Command | What it does |
|---------|-------------|
| `squire refactor rename <old> <new>` | Rename a symbol across the codebase |
| `squire refactor move <sym> <dest>` | Move a symbol to another package |
| `squire refactor propagate <fn> <err>` | Add error return through callers |
| `squire refactor extract <fn> <pkg>` | Extract a function and its private deps to a new package |

All refactor commands default to **dry-run** (preview the diff). Pass `--apply` to modify files.

**Flags:**
- `--apply` — actually modify files (default: dry-run)
- `--include-comments` — also rename in comments
- `--format <unified|json|summary>` — output format
- `--lsp-cmd "gopls serve"` — use LSP for type-aware refactoring (supports gopls, pyright, rust-analyzer, clangd, typescript-language-server)

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

## AI Integration

`squire init` automatically configures your AI tools. It detects which tools you use and installs the right configuration for each:

| AI Tool | What gets installed | How it's detected |
|---------|-------------------|-------------------|
| **Claude Code** | `.claude/skills/squire.md` | `.claude/` directory exists |
| **Cursor** | Appends to `.cursorrules` | `.cursorrules` or `.cursor/` exists |
| **GitHub Copilot** | `.github/copilot-instructions.md` | `.github/` directory exists |
| **Any AI tool** | `.aidocs/AGENTS.md` | Always created |

The skill/rules files teach each AI assistant to:
1. Read `.aidocs/manifest.aid` first for the package map
2. Read a package's `.aid` file before reading its source code
3. Use `squire query` to trace dependencies without reading source
4. Use `squire show` / `squire excerpt` for targeted source reading
5. Use `squire refactor` for precise codebase-wide changes
6. Use `squire impact` / `squire estimate` for change analysis

### Manual installation

If `squire init` doesn't detect your AI tool, you can install skills manually. Templates are in the [`skills/`](skills/) directory:

**Claude Code:**
```bash
mkdir -p .claude/skills
cp skills/claude-skill.md .claude/skills/squire.md
```

**Cursor:**
```bash
cat skills/cursorrules.md >> .cursorrules
```

**GitHub Copilot:**
```bash
cp skills/copilot-instructions.md .github/copilot-instructions.md
```

**Windsurf:**
```bash
cp skills/windsurfrules.md .windsurfrules
```

**Any other tool:** Point it at `.aidocs/AGENTS.md` — it's plain markdown that any LLM can read.

### What the Claude Code skill does

The Claude Code skill (`.claude/skills/squire.md`) triggers automatically when Claude needs to understand code relationships, trace call chains, or refactor. It teaches Claude to:

- Read AID files before source code (reducing token consumption)
- Use `squire show <symbol>` to read just the function you need
- Use `squire query callstack <fn> --up` to find callers instead of grepping
- Use `squire query depends <Type>` to trace impact of changes
- Use `squire refactor rename/move/propagate/extract` for precise codebase-wide changes
- Use `squire impact` to check blast radius before committing
- Run `squire generate` after making changes to keep `.aidocs/` current

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

For detailed claim-level staleness (which specific `[src:]` references are outdated):

```bash
$ squire stale

Found 4 stale claim(s) across 2 AID file(s):

server (server.aid):
  HandleRequest.calls: file changed at server/handler.go:45
  HandleRequest.effects: lines changed at server/handler.go:52

config (config.aid):
  Config.Load.sig: file changed at config/loader.go:20
  Config.Validate.calls: lines changed at config/loader.go:35
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

### Show: read just the function you need
```bash
$ squire show Server.Start

// Start the HTTP server
func (s *Server) Start(addr string) error {
    if err := s.config.Validate(); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }
    ...
}

// Source: server.go:20-45 (internal/server)
```

### Impact: what did you miss?
```bash
$ squire impact

  Changed files: 3
  Changed symbols: 8

  Blast radius:
    Functions affected:  23
    Packages affected:  4 (server, config, handler, middleware)

  ⚠ Potentially missed (5 symbols in 2 files outside your changeset):

    middleware:
      AuthMiddleware (Function)
      RateLimiter (Function)

  Risk: ⚠ MEDIUM
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

### Refactor: type-aware rename with LSP
```bash
$ squire refactor rename Get Fetch --lsp-cmd "gopls serve" --apply

Modified 8 file(s), 23 edit(s) applied, 3 AID file(s) updated
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
  ✓ AID validation: all files pass
  ✓ Coverage: 42/42 packages have AID files

All checks passed. Your project is ready for AI-assisted development.
```

## Architecture

Squire is a single Go binary (~4MB) that embeds:
- The [aidkit](https://github.com/dan-strohschein/aidkit) library — parser, validator (10 spec rules), discovery protocol, and L2 staleness checking
- The [Cartograph](https://github.com/dan-strohschein/Cartograph) graph engine — cached loading (~6x faster via gob serialization), semantic queries, search, and lock support
- The [Chisel](https://github.com/dan-strohschein/chisel) refactoring engine — rename, move, propagate, extract, impact analysis, and lock safety checking

Language-specific generators (`aid-gen-go`, `aid-gen-ts`, etc.) are separate binaries managed by `squire install`.

## License

MIT
