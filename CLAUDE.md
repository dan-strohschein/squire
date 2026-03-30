# Squire

Unified AI code assistant toolkit. Single binary that embeds AID generation, Cartograph (semantic graph), and Chisel (refactoring).

## Architecture

Pipeline architecture — each command is a thin orchestrator over embedded libraries:

- **detect** (`internal/detect/`) — Language detection from file markers (go.mod, package.json, etc.)
- **generate** (`internal/generate/`) — L1 AID generation. Shells out to language-specific generators (aid-gen-go, etc.) and stamps `@code_version` for staleness tracking.
- **carto** (`internal/carto/`) — Embedded Cartograph engine. Imports `cartograph/pkg/{graph,loader,query,output}` directly — no external binary.
- **refactor** (`internal/refactor/`) — Embedded Chisel engine. Implements `resolve.GraphQuerier` using the embedded Cartograph instead of shelling out.
- **agents** (`internal/agents/`) — Generates AGENTS.md with universal AI instructions + project-specific hints.
- **status** (`internal/status/`) — Git-based staleness detection. Compares `@code_version git:<hash>` in .aid files against current HEAD.
- **tools** (`internal/tools/`) — Downloads tool binaries from GitHub releases. Tracks versions in `~/.squire/versions.json`.

## Dependencies

- `github.com/dan-strohschein/aidkit` — AID parser (pkg/parser) for validation and manifest generation
- `github.com/dan-strohschein/cartograph` — Graph engine (pkg/graph, pkg/loader, pkg/query, pkg/output)
- `github.com/dan-strohschein/chisel` — Refactoring engine (resolve, edit, patch)

Local replace directives in go.mod point to sibling directories during development.

## Build

```bash
go build -o squire ./cmd/squire
```

With version stamping:
```bash
go build -ldflags "-X github.com/dan-strohschein/squire/internal/version.Version=v1.0.0" -o squire ./cmd/squire
```

## Key Design Decisions

- **Embed over shell-out** — Cartograph and Chisel are imported as Go libraries, not external binaries. This means `squire init` + `squire query` work with zero additional installs.
- **Generators are external** — Language-specific generators (aid-gen-go, aid-gen-ts, etc.) are separate binaries because they have language-specific dependencies. Squire downloads them on demand.
- **AGENTS.md is the AI integration point** — A plain markdown file that any LLM can read. Not Claude-specific, not Cursor-specific. Universal.
- **Dry-run by default** — All refactor commands preview changes. Pass `--apply` to modify files.
- **Staleness via git hash** — Each .aid file gets `@code_version git:<short-hash>` stamped after generation. `squire status` and `squire generate` compare against current HEAD per package directory.

## Commands

| Command | Entry point | What it does |
|---------|------------|-------------|
| `init` | `cmdInit()` | Detect → Generate → Validate → Graph stats → AGENTS.md |
| `generate` | `cmdGenerate()` | Staleness check → Regenerate stale → Rebuild AGENTS.md |
| `status` | `cmdStatus()` | Show stale/fresh/missing counts + graph stats |
| `doctor` | `cmdDoctor()` | Check tools + project + graph + coverage |
| `query` | `cmdQuery()` → `carto.Run()` | Embedded Cartograph queries |
| `refactor` | `cmdRefactor()` | Intent → Resolve → Edit → Patch (all embedded) |
| `install` | `cmdInstall()` → `tools.Install()` | Download from GitHub releases |
| `upgrade` | `cmdUpgrade()` → `tools.Upgrade()` | Check + update all installed tools |
