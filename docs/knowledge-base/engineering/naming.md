# Naming

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

Names are the primary documentation. A procedure takes a strong verb-noun pair; a function is
named for **what it returns**; a boolean reads as a predicate and is never negated; literals
become named constants at first use. Vague identifiers — `temp`, `data`, `info`, `obj`, `val`,
`result`, `flag`, `x` — are legal only as a short-lived loop index.

## Why it exists

Because code is read far more often than written, and a name is the only documentation that
cannot drift out of step with the code. A comment explaining what a badly named function does
is two artefacts to maintain, one of which is not checked by anything.

The negated-boolean rule exists for a specific mechanical reason: `if !notFound` is a double
negative at every call site, and readers misparse it. The naming rule removes the cost at the
definition instead of at each use.

## When to use it

Every identifier, every time. It is also the first thing to reach for when code feels
unclear — before a comment, before a smaller function. One vocabulary per concept across the
whole codebase: do not use fetch/get/retrieve/load interchangeably for one operation.

## How to use it

```go
// RIGHT — the function is named for what it returns.
func digitRunsIn(s string) []Run

// RIGHT — procedure: strong verb-noun pair.
func recordDecision(ctx context.Context, d Decision) error

// RIGHT — boolean reads as a predicate, stated positively.
func (g Grant) IsLive(now time.Time) bool
type Features struct{ StructuralPolicyEnforced bool }

// RIGHT — the literal has a name at first use.
const maxChangedFiles = 1000
```

**The invariant: a name needing "and" or "or" means the thing must be split.** The name is
reporting a design defect, not a wording problem.

## How not to use it

**Naming for the mechanism instead of the result.**

```go
// WRONG — names neither what it returns nor what it searched.
func numbersIn(s string) []Run
// Symptom: the caller has to read the body to learn what comes back.
```

**Negated booleans.**

```go
// WRONG
type Config struct{ SchedulerDisabled bool }
if !cfg.SchedulerDisabled { run() }      // double negative at every call site
// Symptom: readers pause. Name it SchedulerEnabled and the pause disappears.
```

**Vague identifiers outside a loop index.**

```go
// WRONG — four names, none of which say anything.
func process(data []byte) (result map[string]any, err error) {
    temp := parse(data)
    info := enrich(temp)
    return info, nil
}
// Symptom: every line must be read to learn what the function is about.
```

**Magic values.**

```go
// WRONG — 1000 and "authz" appear with no name and will appear again elsewhere.
if len(c.Paths) > 1000 { return ErrTooManyFiles }
if kind == "authz" { requireReview() }
// Symptom: the second occurrence drifts from the first, silently.
```

**Weak verbs in a procedure name.** `process`, `handle`, `manage`, `doWork` describe nothing.
If no strong verb fits, the routine probably has no single job.

**Trade-off:** good names are longer, and a precise name is harder to choose than a vague one
— the cost is paid once by the author and recovered every time anyone reads it.

**Related:** `routines.md`, `comments.md`, `duplication.md`
