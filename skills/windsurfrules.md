# Windsurf Rules Template for Squire
# Copy this content to .windsurfrules in your project root

This project uses Squire for structured code documentation in `.aidocs/`.

Before reading source code for any package, read its `.aid` file in `.aidocs/` first. AID files contain function signatures (@fn), call graphs (@calls), type definitions (@type), workflows (@workflow), invariants, and antipatterns.

Start with `.aidocs/manifest.aid` for the package index.

Use `squire show <symbol>` to read just a function body instead of entire files.
Use `squire excerpt file.go:Func1,Func2` to extract specific functions from a file.

Use `squire query` to trace dependencies:
- `squire query callstack <function> --up` — find callers
- `squire query callstack <function> --down` — find callees
- `squire query depends <Type>` — find dependents
- `squire query field <Type.Field>` — find what touches a field
- `squire query errors <ErrorType>` — find error producers
- `squire query effects <function>` — find side effects
- `squire query search "<pattern>"` — find by name

Use `squire impact` to check blast radius before pushing changes.
Use `squire estimate <Symbol>` to estimate effort for changes.
Use `squire stale` to check which AID claims have outdated source references.

Use `squire refactor` for precise changes (dry-run by default, --apply to modify):
- `squire refactor rename <old> <new>`
- `squire refactor move <symbol> <package>`
- `squire refactor propagate <function> <error>`
- `squire refactor extract <function> <new-package>`

Run `squire generate` after code changes to update `.aidocs/`.
