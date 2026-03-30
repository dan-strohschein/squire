# Cursor Rules Template for Squire
# Copy this content to .cursorrules in your project root

This project uses Squire (AID + Cartograph + Chisel) for structured code documentation and semantic queries.

## Before Reading Source Code

Check if `.aidocs/` exists. If it does:
1. Read `.aidocs/manifest.aid` for the package index
2. Read the `.aid` file for any package before reading its source — it has function signatures (@fn), call graphs (@calls), workflows (@workflow), invariants, and antipatterns

## Querying the Codebase

Use `squire query` to trace relationships instead of grepping:

- `squire query callstack <function> --up` — who calls this function?
- `squire query callstack <function> --down` — what does this function call?
- `squire query depends <Type>` — what depends on this type?
- `squire query search "<pattern>"` — find by name (glob/regex)
- `squire query list <module>` — list everything in a module

## Refactoring

Use `squire refactor` for precise codebase-wide changes (dry-run by default):

- `squire refactor rename <old> <new>` — rename a symbol everywhere
- `squire refactor move <symbol> <package>` — move between packages
- `squire refactor propagate <fn> <error>` — add error return through callers

Pass `--apply` to actually modify files.

## AID File Quick Reference

| Field | Meaning |
|-------|---------|
| @fn / @sig | Function signature |
| @calls | Functions this calls (for tracing call chains) |
| @type / @fields | Struct with its fields |
| @trait / @requires | Interface contract |
| @workflow / @steps | Multi-step data flow |
| @invariants | Constraints that always hold |
| @antipatterns | Common mistakes to avoid |
| @error_map | Error taxonomy with classification |
| @lock | Mutex documentation with ordering |

## After Making Changes

Run `squire generate` to update `.aidocs/` with your code changes.
