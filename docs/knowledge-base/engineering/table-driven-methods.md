# Table-driven methods

*Construction · Derived from the global `cc2-table-driven` skill — **binding on this project***

## What it is

Replacing branching logic with a data lookup. Complex branching is usually **data wearing a
costume**: a conditional chain testing one value against constants collapses into a named
table plus a trivial lookup. Three access methods — **direct** (key maps straight to an entry),
**indexed** (a sparse key mapped through a small index), and **stair-step** (ranges, walked
until the input fits).

## Why it exists

Because it collapses cyclomatic complexity, makes new cases **additive rather than
structural**, and turns "did we handle every branch?" into a question you can answer by
reading a table. A fifteen-branch switch has fifteen paths a reader must verify; a
fifteen-row table has one path and fifteen rows to scan.

## When to use it

Look for the transformation when the code has:

- A `switch` or `if/else` chain over three or more cases that all do structurally the **same**
  thing with different values.
- Conditionals testing the same expression repeatedly against different constants.
- Nested conditionals selecting a rate, label, fee, severity or handler.
- A chain of range comparisons.

**Do not apply it when the branches do genuinely different work** with different control
flow, side effects or early exits. A table of dissimilar closures is harder to read than the
conditional it replaced. **Complexity moved is not complexity removed.**

## How to use it

Extract the data to a named, module-level constant, keep the lookup trivial, and handle the
miss explicitly.

```go
// RIGHT — direct access. Named for its contents, one row per case.
var severityByKind = map[Kind]Severity{
    KindAuthz:        SeverityHigh,
    KindRoute:        SeverityMedium,
    KindIaC:          SeverityHigh,
    KindCIConfig:     SeverityHigh,
    KindPolicyConfig: SeverityCritical,
}

func severityFor(k Kind) (Severity, error) {
    s, ok := severityByKind[k]
    if !ok {
        return 0, fmt.Errorf("%w: %q", ErrUnknownKind, k)   // the miss is explicit
    }
    return s, nil
}
```

```go
// RIGHT — stair-step for ranges. Ordered ascending, walked until it fits.
var patchSizeTiers = []struct {
    UpTo  int
    Tier  Tier
}{
    {UpTo: 100, Tier: TierSmall},
    {UpTo: 1_000, Tier: TierMedium},
    {UpTo: 10_000, Tier: TierLarge},
}
```

**The invariant: an unmatched key produces a defined result or a deliberate error, never a
silent zero value.** Where the language offers exhaustiveness checking over a closed type,
prefer it — it gives the missing-case guarantee at compile time.

## How not to use it

**A conditional chain left as one.**

```go
// WRONG — five branches, same test, structurally identical bodies.
func severityFor(k Kind) Severity {
    if k == KindAuthz { return SeverityHigh }
    if k == KindRoute { return SeverityMedium }
    if k == KindIaC   { return SeverityHigh }
    if k == KindCIConfig { return SeverityHigh }
    return SeverityLow            // and the miss is silently a value
}
// Symptom: adding a kind is a structural edit, and an unknown kind is reported
// as low severity rather than as unknown.
```

**A silent miss.**

```go
// WRONG — the zero value is a plausible answer.
func severityFor(k Kind) Severity { return severityByKind[k] }
// Symptom: an unmapped kind returns Severity(0). If that happens to be
// SeverityInfo, an unrecognised change is silently judged harmless.
```

**Tabling dissimilar work.**

```go
// WRONG — the branches have different control flow, so the table hides it.
var handlerByKind = map[Kind]func(Change) error{
    KindAuthz: func(c Change) error { /* 20 lines, three early returns */ },
    KindIaC:   func(c Change) error { /* parses HCL, calls out, retries */ },
}
// Symptom: reading the table tells you nothing about what happens. The
// conditional was clearer.
```

**Logic interleaved with the data.** A table with conditionals between the rows, or a lookup
routine that grows branches of its own, means the table is the wrong shape.

**Naming the table for its type.** `kindMap` says what it is; `severityByKind` says what it
holds.

**Trade-off:** the behaviour moves away from the call site into a constant elsewhere in the
file, so tracing one case means two hops instead of one.

**Related:** `control-flow.md`, `naming.md`
