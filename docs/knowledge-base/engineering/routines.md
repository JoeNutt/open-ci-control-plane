# Routines

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

One routine, one job. **Parameters: aim for zero, one or two; three is the working limit.**
Cyclomatic complexity under 10. Parameters ordered input → modify → output, consistently. No
boolean parameter that selects behaviour.

## Why it exists

Arguments are couplings, and they are harder to read than they are to write — the IDE helps
the author and not the reader. Each extra argument with no natural ordering multiplies the
ways to call it wrongly, and none of those ways is caught by the compiler when the types
coincide.

The complexity limit exists because branches multiply the paths a reader must hold
simultaneously, and past roughly ten nobody holds them all.

## When to use it

Writing any routine, and reviewing any that has grown. **The limit is a heuristic, not a
law:** where the ordering has a natural rationale — `distance(x1, y1, x2, y2)` — more than
three is fine, because there is no way to get them wrong. Where there is no such rationale,
a long list is the signal to stop and group.

The grouping test: *if several values are cohesive enough to be passed as a unit, why are they
not already a type?*

## How to use it

```go
// RIGHT — the cohesive values became the type they already were.
type Limits struct {
    MaxFiles      int
    MaxPatchBytes int
    MaxDepth      int
    MaxTotalBytes int
}

func CheckBounds(c Change, l Limits) (Bounded, error)
```

```go
// RIGHT — natural ordering, so four parameters are safe.
func rectangle(x1, y1, x2, y2 int) Rect
```

```go
// RIGHT — two routines rather than one with a selector.
func (g Gate) Decide(c Change) Verdict          // enforcing
func (g Gate) DecideAdvisory(c Change) Verdict  // reporting only
```

**The invariant: a name needing "and" means two routines.** `initAndProcess` is reporting its
own defect.

## How not to use it

**A boolean that selects behaviour.**

```go
// WRONG — unreadable at the call site, and it is two routines.
func (g Gate) Decide(c Change, enforce bool) Verdict
v := gate.Decide(change, true)        // true meaning what?
// Symptom: the reader opens the definition to understand the call.
```

**Parameters past the limit with no rationale.**

```go
// WRONG — seven arguments, no natural ordering, all plausible to transpose.
func Evaluate(c Change, l Limits, caps Capabilities, clock Clock,
              rev string, dryRun bool, tenant string) (Verdict, error)
// Symptom: Evaluate(change, limits, caps, clock, tenant, false, rev) compiles.
// Two of those are swapped.
```

**One routine, two jobs.**

```go
// WRONG — builds the document AND records the decision.
func buildAndRecord(ctx context.Context, e Event) (Document, error) {
    doc := assemble(e)
    if err := db.Save(ctx, doc); err != nil { return Document{}, err }
    return doc, nil
}
// Symptom: the name needs "and", and the function cannot be tested without a store.
```

**Complexity accumulated rather than extracted.** A routine with eleven branches is not made
acceptable by being correct. Extract the interior and name it — `control-flow.md` covers the
mechanics.

**Trade-off:** grouping parameters into types means more types, and each one is a name to
choose and a file to find. That is cheaper than a call site nobody can read.

**Related:** `naming.md`, `control-flow.md`, `classes-and-abstractions.md`
