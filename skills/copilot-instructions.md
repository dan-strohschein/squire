# GitHub Copilot Instructions Template for Squire
# Copy this content to .github/copilot-instructions.md in your project

# Squire — Structured Code Documentation

This project uses Squire for AI-readable code documentation. The `.aidocs/` directory contains structured documentation files that describe every package's API surface, call graph, workflows, and known issues.

## How to Use

1. **Start with the manifest**: `.aidocs/manifest.aid` lists all packages and their relationships
2. **Read AID before source**: For any package, read its `.aid` file first — it has function signatures, call graphs, and semantic annotations
3. **AID files document**: function signatures (@fn, @sig), call relationships (@calls), type definitions (@type, @fields), interfaces (@trait), workflows (@workflow), invariants (@invariants), antipatterns (@antipatterns), error taxonomy (@error_map), and lock ordering (@lock)

## Terminal Commands

If you can execute commands:

### Reading source code
- `squire show <symbol>` — extract just a function/type body by name (saves 80-90% of read tokens)
- `squire excerpt file.go:Func1,Func2` — extract specific functions from a file

### Querying the graph
- `squire query callstack <function> --up` — find all callers of a function
- `squire query callstack <function> --down` — find all callees
- `squire query depends <Type>` — find what depends on a type
- `squire query field <Type.Field>` — find what reads or writes a field
- `squire query errors <ErrorType>` — find where an error is produced
- `squire query effects <function>` — find side effects of a function
- `squire query search "<pattern>"` — find functions/types by name

### Analysis
- `squire impact` — blast radius of uncommitted changes
- `squire impact <Symbol>` — what breaks if this symbol changes
- `squire estimate <Symbol1> <Symbol2>` — estimate effort from graph analysis
- `squire stale` — check which AID claims have outdated source references

### Refactoring (dry-run by default, pass --apply to modify)
- `squire refactor rename <old> <new>` — preview a rename across the codebase
- `squire refactor move <symbol> <package>` — move between packages
- `squire refactor propagate <fn> <error>` — add error return through callers
- `squire refactor extract <fn> <package>` — extract function and deps to new package

### Maintenance
- `squire generate` — regenerate `.aidocs/` after code changes
