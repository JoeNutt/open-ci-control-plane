# Control flow

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

Write the nominal path first, then the unusual and error cases. Guard clauses with early
returns at the top, then a single exit for the main path. **Nesting deeper than three levels
is a defect** — extract the interior and name it. Make execution dependencies explicit by
passing one step's output as the next step's argument.

## Why it exists

Because the main purpose of a block should be visible without scrolling, and accumulated
guard conditions bury it. Arrow-shaped code forces a reader to track which of five
conditions they are inside before they can read the line that matters.

The explicit-dependency rule exists because ordering a reader has to *infer* is temporal
coupling — it compiles, works today, and breaks silently when someone reorders.

## When to use it

Writing any routine with more than one branch, and whenever a block has drifted past three
levels of indentation. Also when a sequence of calls depends on an order nothing enforces.

## How to use it

```go
// RIGHT — guards first, nominal path flat and last. Nothing nested.
func (g Gate) Decide(ctx context.Context, c Change) (Verdict, error) {
    if !c.WellFormed() {
        return Verdict{}, ErrMalformedChange
    }
    if c.Totals.Files == 0 {
        return Verdict{Effect: Admit}, nil
    }

    violations := g.evaluate(c)
    return Verdict{Effect: effectFor(violations), Reasons: violations}, nil
}
```

```go
// RIGHT — the dependency is in the signature, so the order cannot be got wrong.
bounded, err := CheckBounds(raw, limits)
if err != nil { return err }
verdict := Evaluate(bounded)            // cannot be called with `raw`
```

```go
// RIGHT — DeMorgan applied, so the condition reads once.
if !(hasTest && hasReview) { requireSignOff() }
```

**The invariant: no routine nests more than three levels.** When it does, the interior is a
routine that has not been named yet.

## How not to use it

**Arrow code from accumulated guards.**

```go
// WRONG — five levels. The line that matters is invisible.
func handle(ctx context.Context, e Event) error {
    if e.Valid() {
        if e.PR != nil {
            if len(e.PR.Files) > 0 {
                if !e.PR.Draft {
                    if cfg.Enabled {
                        return process(ctx, e)      // the actual work, buried
                    }
                }
            }
        }
    }
    return nil        // and every failure returns success
}
// Symptom: indentation depth 5, and the implicit else returns nil for every
// rejection — four distinct refusals reported as success.
```

**Implicit ordering.**

```go
// WRONG — Evaluate depends on CheckBounds having run. Nothing says so.
CheckBounds(raw, limits)        // return value discarded
verdict := Evaluate(raw)        // reads a field CheckBounds was supposed to set
// Symptom: reordering two lines changes behaviour with no compile error.
```

**Scattered returns through a long routine.** Multiple returns are justified where they beat
nesting — at the top, as guards. Returns sprinkled through forty lines mean no reader can
tell what the exit conditions are.

**A conditional chain over one value.**

```go
// WRONG — five branches testing the same expression against constants.
if kind == "authz" { return SeverityHigh }
if kind == "route" { return SeverityMedium }
if kind == "iac"   { return SeverityHigh }
// Symptom: adding a kind is a structural edit. This is a table — see
// table-driven-methods.md.
```

**Breaking arbitrarily from the middle of a loop**, or recursion spanning routines. A cyclic
recursion chain across functions is close to impossible to trace; keep recursion within one
routine and to genuinely hierarchical data.

**Trade-off:** guard clauses mean more exit points, which is worse for anyone tracing a single
path and much better for everyone reading the routine's shape.

**Related:** `routines.md`, `table-driven-methods.md`, `errors-and-assertions.md`
