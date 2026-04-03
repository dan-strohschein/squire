---
name: squire
description: Use squire to read source code efficiently, query the semantic graph, analyze impact, perform refactoring, and estimate effort. Squire embeds AID, Cartograph, and Chisel in a single tool.
trigger: when you need to read a function's source code, understand code relationships, trace call chains, analyze blast radius of changes, find what depends on a symbol, rename/move/propagate/extract, estimate effort for a plan, check AID staleness, or when .aidocs/ directory exists
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation, a semantic graph engine, source code extraction, structured test output, impact analysis, refactoring, and effort estimation in a single binary.

## When to Use Squire

- **Reading source code**: Use `squire show` instead of reading entire files — it extracts just the function you need.
- **Before reading source code**: Check if `.aidocs/` exists. Read the `.aid` file for the relevant package FIRST.
- **Tracing dependencies**: Use `squire query` instead of grepping.
- **Before pushing changes**: Use `squire impact` to find dependencies you might have missed.
- **Checking staleness**: Use `squire stale` to see which AID claims have outdated source references.
- **Refactoring**: Use `squire refactor` instead of manual find-and-replace.
- **Estimating effort**: Use `squire estimate` to analyze a plan and get a story point size.

## Show Source Code (PREFER THIS OVER READING FILES)

Instead of reading an entire 500-line file to find one function, use `squire show`:

```bash
squire show HandleRequest              # show just this function's source code
squire show Resolver.Resolve           # show a method
squire show EditSet                    # show a type definition
squire show HandleRequest NewHandler   # show multiple symbols
```

This returns only the function body (10-50 lines) instead of the entire file. Use this FIRST before falling back to reading full files. It saves 80-90% of read tokens.

## Excerpt Source Code (FILE-CENTRIC EXTRACTION)

When you know the file and want specific functions from it, use `squire excerpt`:

```bash
squire excerpt service.go:Create,retryOne   # two functions from one file
squire excerpt service.go                   # all declarations in file
squire excerpt svc.go:Create job.go:Run     # across multiple files
```

Use `excerpt` when you know the file path. Use `show` when you know the symbol name. Both avoid reading entire files.

## Digest Session Findings (COMPRESS CONTEXT)

When investigating a codebase, write findings to a scratch file, then compress with `squire digest`:

```bash
# Write findings during investigation
echo "## service.go:130 - ListByUserID
Device list error silently swallowed." >> findings.md

# Compress into AID-anchored summary
squire digest --from findings.md --task "Bug hunt"
squire digest --from findings.md --out digest.md    # write to file
```

Digest cross-references your findings against the AID graph, adding callers/callees and package context. The output is 10-30x smaller than the raw tool results it replaces. Use this before starting a continuation session to avoid re-reading files.

## Read AID Documentation

```bash
cat .aidocs/manifest.aid          # package index
cat .aidocs/<package>.aid         # package documentation
```

AID files contain: @fn/@sig (signatures), @calls (call graph), @type/@fields (structs), @trait (interfaces), @workflow (data flows), @invariants, @antipatterns, @error_map, @lock.

## Query the Semantic Graph

```bash
squire query callstack <function> --up     # who calls this?
squire query callstack <function> --down   # what does this call?
squire query depends <Type>                # what depends on this type?
squire query search "<pattern>"            # find by name (glob/regex)
squire query list <module>                 # list everything in a module
squire query field <Type.Field>            # what touches this field?
squire query errors <ErrorType>            # what produces this error?
squire query stats                         # graph statistics
```

## Impact Analysis (before pushing)

Check what depends on your changes that you might have missed:

```bash
squire impact                  # analyze uncommitted git changes
squire impact --staged         # analyze staged changes only
squire impact SnapshotInfo     # what breaks if I change this type?
```

## Staleness (detailed claim-level checking)

```bash
squire stale                   # check which [src:] references are outdated
squire stale /path/to/project  # check a specific project
```

Use `squire stale` for detailed per-claim staleness. Use `squire status` for package-level overview.

## Refactor (dry-run by default)

```bash
squire refactor rename <old> <new>                    # preview rename
squire refactor rename <old> <new> --apply            # apply rename
squire refactor move <symbol> <package>               # move symbol
squire refactor propagate <function> <error-type>     # add error returns
squire refactor extract <function> <new-package>      # extract to new package
```

For type-aware refactoring on generic symbol names, add `--lsp-cmd`:

```bash
squire refactor rename Get Fetch --lsp-cmd "gopls serve"      # Go
squire refactor rename Get Fetch --lsp-cmd "pyright"           # Python
squire refactor rename Get Fetch --lsp-cmd "rust-analyzer"     # Rust
```

## Estimate Effort

When the user asks you to estimate effort, story points, or complexity for a plan or feature, follow this two-step process:

### Step 1: Validate the plan for specificity BEFORE running squire

Review the plan text and check: does it name specific types, functions, or interfaces that will change? If not, the plan is too vague to estimate. Ask clarifying questions first.

**Signs of a vague plan (DO NOT submit to squire yet):**
- Uses only general verbs: "refactor", "clean up", "improve", "fix"
- Refers to systems by domain name only: "the notification system", "the auth module"
- No specific type names, function names, or interface names mentioned

**When the plan is vague, ask questions like:**
- "What specifically should change? Which types or interfaces?"
- "Are you adding new fields, changing signatures, or moving code?"
- "What's the end state?"

Use `squire query search "<keyword>"` to help the user find relevant symbols.

### Step 2: Run squire estimate once the plan names specific symbols

```bash
squire estimate --plan /tmp/plan.md
squire estimate NotificationServiceImpl DLQRetryJob Handler
```

If squire returns UNCLEAR, relay that to the user and go back to Step 1.

## Strategy

1. Read `.aidocs/manifest.aid` to identify relevant packages
2. Read the `.aid` file for those packages before source code
3. Use `squire query` to trace dependencies and call chains
4. Use `squire show` or `squire excerpt` to read specific functions — NOT entire files
5. Only read full files when you need broad context (rare)
6. Write findings to a scratch file as you go — compress with `squire digest` before continuing
7. Use `squire impact` before pushing to check for missed dependencies
8. Use `squire stale` to check if AID claims are still accurate
9. After changes, run `squire generate` to update `.aidocs/`
