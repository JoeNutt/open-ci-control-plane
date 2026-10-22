# Deployability

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

How easily, safely and often a change can be released. Treated as an **architectural
property** rather than a pipeline concern: a design that forces coordinated releases across
five modules has made a structural choice, whatever the CI configuration says.

## Why it exists

Because slow releases mean slow feedback, and slow feedback degrades every decision that
follows. If a change takes two weeks to reach production, the information about whether it was
right arrives two weeks late, by which time three more decisions have been built on it.

It is a separate principle from testability because it fails differently: a design can be
perfectly testable and still undeployable, held together by a release train nobody chose.

## When to use it

As a design test, alongside testability. Ask of any boundary: **can these be built, tested,
released and run independently?** Where the answer is no, name which kind of coupling is
stopping it — developmental (cannot release until you finish) or operational (cannot start
unless you are running). Both are design choices, not facts.

Also when weighing a split: independent deployability is one of the few legitimate reasons to
cross a process boundary — and it has a cost most people skip.

## How to use it

Keep changes small, independently releasable and reversible, and make the reversibility real.

```go
// RIGHT — a change gated behind a declared, revertible switch, so release and
// activation are separate events and rollback is a config change not a deploy.
type Features struct {
    StructuralPolicyEnforced bool   // ship at false, enable per repository
}

func (g Gate) Decide(c Change, f Features) Verdict {
    v := g.evaluate(c)
    if !f.StructuralPolicyEnforced {
        return v.AsAdvisory()        // same decision, no enforcement
    }
    return v
}
```

**The invariant: the ability to release one module without releasing the others is a property
you can demonstrate, not assert.** If demonstrating it requires a coordinated test of all five,
they are one deployable unit.

## How not to use it

**A shared release train disguised as independent services.**

```go
// WRONG — two services, one version number.
// policy v2.4.0 requires matrix v2.4.0 exactly; the contract is implicit.
type EvaluateRequest struct {
    Change Change   // marshalled with encoding/gob, no version field
}
// Symptom: beautifully decoupled at code level, developmentally coupled by
// release. Neither can ship alone, so the split bought latency and nothing else.
```

**Treating independent deployment as free.** The specific cost: **services that deploy
independently are not tested together.** You trade release autonomy for the guarantee that the
combination works. That is often the right trade — but it is a trade, and it has to be made
deliberately rather than discovered after the first incompatible pair reaches production.

**A change that cannot be reverted.** A migration that drops a column, a message format with
no version field, a cache key change with no fallback — each makes rollback a forward-fix.
Prefer the decision you can reverse; see `incremental-architecture.md`.

**Optimising the pipeline instead of the design.** Faster CI on a design that requires five
coordinated releases makes the coordinated release faster. The structural problem is
untouched.

**Trade-off:** independently deployable units need versioned contracts, compatibility windows
and more integration testing than a single unit — real, ongoing work bought in exchange for
release autonomy.

**Related:** `module-boundaries.md`, `coupling.md`, `incremental-architecture.md`
