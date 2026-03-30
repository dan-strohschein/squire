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
- `squire query callstack <function> --up` — find all callers of a function
- `squire query callstack <function> --down` — find all callees
- `squire query depends <Type>` — find what depends on a type
- `squire query search "<pattern>"` — find functions/types by name
- `squire refactor rename <old> <new>` — preview a rename across the codebase
- `squire generate` — regenerate `.aidocs/` after code changes
