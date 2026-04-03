# Cursor Rules Template for Squire
# Copy this content to .cursorrules in your project root

This project uses Squire (AID + Cartograph + Chisel) for structured code documentation and semantic queries.

## Before Reading Source Code

Check if `.aidocs/` exists. If it does:
1. Read `.aidocs/manifest.aid` for the package index
2. Read the `.aid` file for any package before reading its source — it has function signatures (@fn), call graphs (@calls), workflows (@workflow), invariants, and antipatterns

## Reading Source Code

Use squire to read only what you need instead of entire files:

- `squire show <symbol>` — extract a function/type body by name (PREFER THIS)
- `squire show Sym1 Sym2` — show multiple symbols at once
- `squire excerpt file.go:Func1,Func2` — extract specific functions from a file
- `squire excerpt file.go` — all declarations in a file

## Querying the Codebase

Use `squire query` to trace relationships instead of grepping:

- `squire query callstack <function> --up` — who calls this function?
- `squire query callstack <function> --down` — what does this function call?
- `squire query depends <Type>` — what depends on this type?
- `squire query field <Type.Field>` — what reads or writes this field?
- `squire query errors <ErrorType>` — what produces this error?
- `squire query effects <function>` — what side effects does this function have?
- `squire query search "<pattern>"` — find by name (glob/regex)
- `squire query list <module>` — list everything in a module

## Analysis

- `squire impact` — blast radius of uncommitted changes (what you might have missed)
- `squire impact --staged` — analyze staged changes only
- `squire impact <Symbol>` — what breaks if you change this symbol?
- `squire estimate <Symbol1> <Symbol2>` — estimate effort from graph analysis
- `squire estimate --plan <file>` — estimate from a plan file
- `squire stale` — check which AID claims have outdated source references

## Refactoring

Use `squire refactor` for precise codebase-wide changes (dry-run by default):

- `squire refactor rename <old> <new>` — rename a symbol everywhere
- `squire refactor move <symbol> <package>` — move between packages
- `squire refactor propagate <fn> <error>` — add error return through callers
- `squire refactor extract <fn> <new-package>` — extract function and deps to new package

Pass `--apply` to actually modify files. Add `--lsp-cmd "gopls serve"` for type-aware refactoring.

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
