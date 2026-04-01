---
name: squire
description: Use squire to read source code efficiently, query the semantic graph, run tests, analyze impact, perform refactoring, and estimate effort. Squire embeds AID, Cartograph, and Chisel in a single tool.
trigger: when you need to read a function's source code, understand code relationships, trace call chains, run tests, analyze blast radius of changes, find what depends on a symbol, rename/move/propagate, estimate effort for a plan, or when .aidocs/ directory exists
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation, a semantic graph engine, source code extraction, structured test output, impact analysis, refactoring, and effort estimation in a single binary.

## When to Use Squire

- **Reading source code**: Use `squire show` instead of reading entire files — it extracts just the function you need.
- **Before reading source code**: Check if `.aidocs/` exists. Read the `.aid` file for the relevant package FIRST.
- **Tracing dependencies**: Use `squire query` instead of grepping.
- **Running tests**: Use `squire test` to get only structured failures, not raw stdout.
- **Before pushing changes**: Use `squire impact` to find dependencies you might have missed.
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

## Run Tests (structured output)

Use `squire test` instead of running test commands directly. It returns only failures with assertion messages and source references — not hundreds of lines of raw output.

```bash
squire test                    # run all tests, show structured failures
squire test ./specific/pkg     # run tests for one package
```

## Impact Analysis (before pushing)

Check what depends on your changes that you might have missed:

```bash
squire impact                  # analyze uncommitted git changes
squire impact --staged         # analyze staged changes only
squire impact SnapshotInfo     # what breaks if I change this type?
```

## Refactor (dry-run by default)

```bash
squire refactor rename <old> <new>                    # preview rename
squire refactor rename <old> <new> --apply            # apply rename
squire refactor move <symbol> <package>               # move symbol
squire refactor propagate <function> <error-type>     # add error returns
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
4. Use `squire show` to read specific functions — NOT entire files
5. Only read full files when you need broad context (rare)
6. Use `squire test` to run tests — parse the structured output, not raw stdout
7. Use `squire impact` before pushing to check for missed dependencies
8. After changes, run `squire generate` to update `.aidocs/`
