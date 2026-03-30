---
name: squire
description: Use squire to query the semantic code graph, understand symbol relationships, and perform precise refactoring. Squire embeds AID documentation, Cartograph graph queries, and Chisel refactoring in a single tool.
trigger: when you need to understand code relationships, trace call chains, find what depends on a symbol, rename/move/propagate changes across a codebase, or when .aidocs/ directory exists in the project
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation (AID files), a semantic graph query engine (Cartograph), and precise refactoring (Chisel) in a single binary.

## When to Use Squire

- **Before reading source code**: Check if `.aidocs/` exists. If it does, read the `.aid` file for the relevant package FIRST — it has function signatures, call graphs, workflows, and known issues.
- **Tracing dependencies**: Use `squire query` instead of grepping. It's faster and gives you the full call chain.
- **Refactoring**: Use `squire refactor` instead of manual find-and-replace. It understands symbol scope and updates AID files too.

## Step 1: Read AID Documentation

If the project has `.aidocs/`, always start here:

```bash
# Read the package index
cat .aidocs/manifest.aid

# Read documentation for a specific package
cat .aidocs/<package>.aid
```

AID files contain:
- `@fn` / `@sig` — function signatures and parameters
- `@calls` — what each function calls (the call graph)
- `@type` / `@fields` — struct definitions
- `@trait` / `@requires` — interface contracts
- `@workflow` — multi-step data flows with numbered steps
- `@invariants` — constraints that always hold (with source references)
- `@antipatterns` — common mistakes to avoid
- `@error_map` — error taxonomy with classification and caller behavior
- `@lock` — mutex documentation with ordering constraints

## Step 2: Query the Semantic Graph

Use `squire query` to trace relationships without reading source:

### Find callers of a function
```bash
squire query callstack <function> --up
```
Example: `squire query callstack HandleRequest --up` shows every function that calls HandleRequest, transitively.

### Find what a function calls
```bash
squire query callstack <function> --down
```

### Find what depends on a type
```bash
squire query depends <Type>
```
Example: `squire query depends SnapshotInfo` finds every function that accepts, returns, or references SnapshotInfo.

### Search by name
```bash
squire query search "<pattern>"
```
Supports glob (`*`) and regex. Example: `squire query search "Handle*"` finds all functions/types matching the pattern.

### List everything in a module
```bash
squire query list <module>
```

### Find what touches a field
```bash
squire query field <Type.Field>
```

### Find error producers
```bash
squire query errors <ErrorType>
```

### Graph statistics
```bash
squire query stats
```

## Step 3: Refactor with Precision

All refactor commands default to **dry-run** (show diff without modifying). Pass `--apply` to make changes.

### Rename a symbol across the codebase
```bash
squire refactor rename <old-name> <new-name>           # preview
squire refactor rename <old-name> <new-name> --apply   # apply
```
Renames the symbol in source code AND updates `.aidocs/` files.

### Move a symbol to another package
```bash
squire refactor move <symbol> <destination-package>
squire refactor move <symbol> <destination-package> --apply
```

### Propagate error returns through callers
```bash
squire refactor propagate <function> <error-type>
squire refactor propagate <function> <error-type> --apply
```

## Strategy: AID-First Development

When working on a task:

1. **Identify relevant packages** from `.aidocs/manifest.aid`
2. **Read the AID files** for those packages — understand the API surface, workflows, and known issues
3. **Use `squire query`** to trace dependencies and call chains as needed
4. **Read source files** only for implementation details not covered by AID
5. **After making changes**, run `squire generate` to update `.aidocs/`

This approach typically reduces the source code you need to read by 50-80%.
