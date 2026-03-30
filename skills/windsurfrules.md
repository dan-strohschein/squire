# Windsurf Rules Template for Squire
# Copy this content to .windsurfrules in your project root

This project uses Squire for structured code documentation in `.aidocs/`.

Before reading source code for any package, read its `.aid` file in `.aidocs/` first. AID files contain function signatures (@fn), call graphs (@calls), type definitions (@type), workflows (@workflow), invariants, and antipatterns.

Start with `.aidocs/manifest.aid` for the package index.

Use `squire query` to trace dependencies:
- `squire query callstack <function> --up` — find callers
- `squire query callstack <function> --down` — find callees
- `squire query depends <Type>` — find dependents
- `squire query search "<pattern>"` — find by name

Use `squire refactor` for precise changes (dry-run by default, --apply to modify):
- `squire refactor rename <old> <new>`
- `squire refactor move <symbol> <package>`
- `squire refactor propagate <function> <error>`

Run `squire generate` after code changes to update `.aidocs/`.
