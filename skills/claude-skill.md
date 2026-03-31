---
name: squire
description: Use squire to query the semantic code graph, understand symbol relationships, perform precise refactoring, and estimate implementation effort. Squire embeds AID documentation, Cartograph graph queries, and Chisel refactoring in a single tool.
trigger: when you need to understand code relationships, trace call chains, find what depends on a symbol, rename/move/propagate changes across a codebase, estimate story points or effort for a plan, or when .aidocs/ directory exists in the project
---

# Squire — AI Code Assistant Toolkit

Squire provides structured codebase documentation (AID files), a semantic graph query engine (Cartograph), precise refactoring (Chisel), and effort estimation in a single binary.

## When to Use Squire

- **Before reading source code**: Check if .aidocs/ exists. If it does, read the .aid file for the relevant package FIRST.
- **Tracing dependencies**: Use squire query instead of grepping.
- **Refactoring**: Use squire refactor instead of manual find-and-replace.
- **Estimating effort**: Use squire estimate — but validate the plan first (see below).

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
- Could mean many different things depending on interpretation

**When the plan is vague, ask questions like:**
- "You said 'refactor the notification system' — what specifically should change? Are you restructuring the package layout, changing the NotificationService interface, adding new methods, or cleaning up implementations?"
- "Which types or services are affected? For example, NotificationServiceImpl, DLQRetryJob, PreferencesService?"
- "Are you adding new fields, changing function signatures, or moving code between packages?"
- "What's the end state? What should be different after this work is done?"

Use `squire query search "<keyword>"` to help the user find relevant symbols:
```bash
squire query search "Notification*"    # find notification-related symbols
squire query list service              # list everything in the service module
```

Share the results with the user to help them name the specific things that will change.

### Step 2: Run squire estimate once the plan names specific symbols

Once the plan references concrete code symbols, run the estimate:

```bash
# Option A: Write the plan to a temp file
squire estimate --plan /tmp/plan.md

# Option B: Pass symbols directly
squire estimate NotificationServiceImpl DLQRetryJob Handler

# Machine-readable output:
squire estimate --plan /tmp/plan.md --format json
```

Report the result to the user including:
- The T-shirt size (TINY/SMALL/MEDIUM/LARGE/XLARGE)
- Number of affected files, functions, and packages
- Any cross-cutting concerns (locks, error maps, antipatterns)
- Specific complexity factors if present

If squire returns UNCLEAR, relay that to the user and go back to Step 1.

## Strategy

1. Read .aidocs/manifest.aid to identify relevant packages
2. Read the .aid file for each package before source code
3. Use squire query to trace dependencies
4. Read source only for implementation details not in AID
5. After changes, run squire generate to update .aidocs/
