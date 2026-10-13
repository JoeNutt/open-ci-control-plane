# Incremental architecture

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

Treating structure as something arrived at by small, reversible steps with feedback, rather
than decided once up front. Three practices: **prefer the decision you can reverse**, **defer
what you can** behind a port, and **treat a structural change as an experiment with a stated
expectation** — then check it.

## Why it exists

Because the information needed to make a structural decision well mostly arrives *after* the
decision has to be made. Deciding everything up front means deciding it with the least
information you will ever have, and then defending it.

It also exists because of a specific failure: a design that cannot be changed incrementally
has already failed the property it most needed. Architecture is the set of decisions expensive
to reverse — so the discipline is to keep as few of them as possible in that category.

## When to use it

- Whenever a choice can be postponed behind an interface without blocking progress — the
  database, the forge provider, the transport.
- Whenever a structural change is proposed: state what you expect to improve, then look.
- Whenever a refactor is too large for the current cycle. It does not get skipped; it gets
  recorded, finished as its own change from green, and committed separately.

## How to use it

Defer the choice, and make the deferral explicit rather than accidental.

```go
// RIGHT — the storage decision is deferred behind a port. The domain works today
// with the in-memory implementation; nothing has to be decided yet.
type DecisionStore interface {
    Save(ctx context.Context, d Decision) error
    Get(ctx context.Context, id DecisionID) (Decision, error)
}

type memoryStore struct{ byID map[DecisionID]Decision }   // ships first
// sqlStore arrives when there is evidence the memory one is insufficient.
```

```go
// RIGHT — a structural change with a stated expectation, so it can be judged.
// "Splitting bounds out of change should make the gate testable without a forge
//  fake, and reduce change's imports from 7 to 3."
// Checked after: imports 7 -> 3, gate test no longer constructs a fake. Kept.
```

**The invariant: judge structure by evidence, not by conformance.** Is change actually getting
easier, are tests getting simpler, is the dependency graph getting cleaner? *"It follows the
pattern"* is not evidence.

## How not to use it

**Designing the whole structure before the first feature.**

```go
// WRONG — eleven packages, two features, no users.
internal/{domain,application,adapters,ports,dto,mappers,factories,
          registry,config,telemetry,shared}/
// Symptom: `shared/` already exists. The structure was derived from a diagram
// rather than from pressure, and every boundary is a guess defended by sunk cost.
```

**A one-way door taken casually.**

```go
// WRONG — the wire format has no version field, so it cannot change.
type Document struct {
    Paths []string `json:"paths"`
}
// Symptom: the first consumer pins this shape forever. A schema_version field
// costs one int now and is impossible to add later. Prefer the reversible choice.
```

**Pausing for a rewrite instead of refactoring toward the boundary.** Refactor incrementally
with tests green. A big-bang restructure has no safe stopping point, so it either lands whole
or is abandoned whole.

**Smuggling a large refactor into a feature change.**

```go
// WRONG — one commit: new behaviour plus a package split plus renames.
// Symptom: unreviewable. Nobody can tell which edits were meant to change
// behaviour and which were not, so nobody can review either half.
```

Finish the cycle, record the larger refactor in the terms of whichever standard it violates,
then do it as its own change from green. If it needs a decision — a boundary moving, an
interface others depend on — surface it rather than deciding unilaterally.

**Trade-off:** deferring costs a port that may turn out to have one implementation forever, and
incremental change means living with an imperfect structure in the meantime. `proportionality.md`
is the counterweight — defer behind a port only where the choice is genuinely live.

**Related:** `proportionality.md`, `deployability.md`,
`../engineering/tdd-red-green-refactor.md`
