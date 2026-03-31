# Plan: `squire estimate` — Story Point Estimation from Graph Analysis

## Context

Developers need to estimate effort before starting work, but estimation is often gut-feel. Squire already has a complete semantic graph of the codebase — function call chains, type dependencies, field access patterns, lock annotations, error maps. This data directly maps to implementation complexity. A `squire estimate` command analyzes an implementation plan against the graph and produces a calibrated T-shirt size estimate (tiny/small/medium/large/xlarge).

**Branch:** `feature/estimate`

---

## How It Works

```
$ squire estimate --plan plan.md

Analyzing implementation plan...

  Symbols identified: SnapshotInfo, IsVisibleToSnapshot, SetSnapshot, GracePeriodMs

  Impact analysis:
    Files to modify:      12
    Packages affected:    6
    Functions to change:  18
    Test files affected:  4
    Cross-cutting:        1 @lock, 1 @error_map, 2 @antipatterns

  Complexity factors:
    ✓ Struct change propagates to 8 construction sites
    ✓ MVCC visibility is performance-critical (@invariant)
    ⚠ GracePeriodMs removal has documented TOCTOU risk (@antipattern)

  Estimate: LARGE (12 files, 18 functions, 6 packages, has cross-cutting concerns)
```

---

## Estimation Algorithm

### Step 1: Extract Target Symbols

Parse the plan text for identifiable symbol names. Two modes:

**a) Explicit symbols via `--symbols` flag:**
```bash
squire estimate --symbols "SnapshotInfo,IsVisibleToSnapshot,SetSnapshot"
```

**b) Extract from plan file via `--plan` flag:**
```bash
squire estimate --plan plan.md
```

For plan files, extract symbols by:
- Finding words that match node names in the graph (case-sensitive match against all graph nodes)
- Prioritizing capitalized words (Go exported names)
- Filtering out common English words and noise

### Step 2: Graph Fan-Out Analysis

For each identified symbol, run the appropriate graph query:

| Symbol kind | Query | What it measures |
|-------------|-------|-----------------|
| Type | `TypeDependents(name)` | Functions that accept/return/reference this type |
| Function/Method | `CallStack(name, Reverse)` | All callers (transitive) |
| Field (Type.Field) | `FieldTouchers(type, field)` | Functions that read/write this field |

Collect into aggregate sets:
- `affectedNodes` — unique function/method nodes across all queries
- `affectedFiles` — unique source files (from `node.SourceFile`)
- `affectedModules` — unique packages (from `node.Module`)

### Step 3: Annotation Scanning

For each affected module, scan the .aid file for:
- `@lock` blocks — cross-cutting concurrency concern (adds complexity)
- `@error_map` blocks — error handling chains that may need updating
- `@antipattern` blocks — known gotchas in affected code
- `@invariant` blocks — constraints that must be preserved

Count these as "complexity factors."

### Step 4: T-Shirt Sizing

Apply sizing rules based on collected metrics:

```
TINY:    files <= 2  AND functions <= 5   AND packages <= 1  AND no cross-cutting
SMALL:   files <= 5  AND functions <= 12  AND packages <= 2  AND cross-cutting <= 1
MEDIUM:  files <= 12 AND functions <= 25  AND packages <= 4  AND cross-cutting <= 2
LARGE:   files <= 25 AND functions <= 50  AND packages <= 8
XLARGE:  anything larger
```

These thresholds can be calibrated from real project data by adding a `squire estimate --calibrate` mode that compares past estimates against actual outcomes.

---

## Implementation

### Files to Create

**`internal/estimate/estimate.go`** — Core estimation logic:

```go
package estimate

type EstimateResult struct {
    // Identified symbols
    Symbols     []SymbolMatch

    // Impact metrics
    Files       int
    Functions   int
    Packages    int
    TestFiles   int

    // Cross-cutting concerns
    Locks       int
    ErrorMaps   int
    Antipatterns int
    Invariants  int

    // Sizing
    Size        string  // "TINY", "SMALL", "MEDIUM", "LARGE", "XLARGE"

    // Detail
    AffectedModules []string
    ComplexityFactors []string
}

type SymbolMatch struct {
    Name     string
    Kind     string  // "type", "function", "method", "field"
    Module   string
    NodeID   string
}

func Estimate(aidDir string, symbols []string) (*EstimateResult, error)
func EstimateFromPlan(aidDir string, planText string) (*EstimateResult, error)
func ExtractSymbols(planText string, graphNodes []string) []string
```

**`internal/estimate/extract.go`** — Symbol extraction from plan text:

```go
func ExtractSymbols(planText string, graphNodeNames []string) []string
```

Builds a set of all node names from the graph, then scans the plan text for matches. Filters out common words, prefers capitalized identifiers, deduplicates.

**`internal/estimate/sizing.go`** — T-shirt size calculation:

```go
func ComputeSize(result *EstimateResult) string
```

### Files to Modify

**`cmd/squire/main.go`** — Add `case "estimate": cmdEstimate(args)` and the `cmdEstimate` function. Accepts:
- `--plan <file>` — read plan from a file
- `--symbols "Sym1,Sym2"` — explicit symbol list
- `--format json` — machine-readable output
- Bare args: `squire estimate SnapshotInfo IsVisibleToSnapshot` — symbols as positional args

### Existing Code to Reuse

| Package | What to use | For |
|---------|------------|-----|
| `cartograph/pkg/loader` | `LoadFromDirectory()` | Load the graph |
| `cartograph/pkg/query` | `QueryEngine` — `CallStack`, `TypeDependents`, `FieldTouchers` | Fan-out analysis |
| `cartograph/pkg/graph` | `Node`, `NodeKind`, `AllNodes()` | Symbol matching, node enumeration |
| `aidkit/pkg/parser` | `ParseFile()` | Scan .aid files for @lock, @error_map, etc. |
| `internal/detect` | `Detect()`, `FindAidocs()` | Project detection |

---

## Output Format

### Human-readable (default)
```
Analyzing plan...

  Symbols found: 4
    SnapshotInfo (Type, models)
    IsVisibleToSnapshot (Function, document)
    SetSnapshot (Method, documentscanner)
    GracePeriodMs (referenced in plan, not in graph)

  Impact:
    Files to modify:      12
    Functions to change:  18
    Packages affected:    6 (models, bundle, planner, server, documentscanner, compactor)
    Test files:           4

  Cross-cutting concerns:
    ⚠ 1 @lock (pageCacheShard.mu — acquired during visibility checks)
    ⚠ 1 @error_map (MVCC errors — may need new error variant)
    ⚠ 2 @antipatterns in affected code

  Estimate: LARGE
```

### JSON (--format json)
```json
{
  "symbols": [...],
  "files": 12,
  "functions": 18,
  "packages": 6,
  "test_files": 4,
  "locks": 1,
  "error_maps": 1,
  "antipatterns": 2,
  "size": "LARGE",
  "affected_modules": ["models", "bundle", "planner", "server", "documentscanner", "compactor"],
  "complexity_factors": [
    "Struct change propagates to 8 construction sites",
    "pageCacheShard.mu lock in affected code path",
    "MVCC visibility is performance-critical (invariant)"
  ]
}
```

---

## Implementation Order

| Step | What | Files |
|------|------|-------|
| 1 | Create branch `feature/estimate` | — |
| 2 | Implement symbol extraction from graph + plan text | `internal/estimate/extract.go` |
| 3 | Implement graph fan-out analysis | `internal/estimate/estimate.go` |
| 4 | Implement annotation scanning for cross-cutting concerns | `internal/estimate/estimate.go` |
| 5 | Implement T-shirt sizing | `internal/estimate/sizing.go` |
| 6 | Wire up CLI command | `cmd/squire/main.go` |
| 7 | Test on proofgo/backend and chisel | — |
| 8 | Test with the SyndrDB SnapshotInfo plan from BM9 | — |

---

## Verification

1. **chisel project:** `squire estimate --symbols "EditKind,GenerateEdits"` → should report SMALL (2-3 files, ~10 functions, 2 packages)
2. **proofgo/backend:** `squire estimate --symbols "NotificationService,Handler"` → should report MEDIUM-LARGE (multiple handler packages, service interfaces)
3. **Plan file mode:** Write a short plan.md describing "rename PlaceholderService to StubService" → `squire estimate --plan plan.md` should identify the symbol and report MEDIUM
4. **JSON output:** Verify `--format json` produces valid parseable JSON
5. **Unknown symbols:** `squire estimate --symbols "FooBarBaz"` → should report "symbol not found in graph" gracefully
