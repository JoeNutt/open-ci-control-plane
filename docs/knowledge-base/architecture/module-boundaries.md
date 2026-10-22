# Module boundaries

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

Where one module ends and the next begins, and separately, whether that line should also be a
**process** boundary. The two questions are different and conflating them is the expensive
mistake.

## Why it exists

Because boundaries drawn in the wrong place cost more than no boundaries at all — they add
ceremony without isolation, and they are hard to move once code depends on them. And because
"split it into services" is routinely offered as the answer to a design problem it cannot
solve.

## When to use it

Draw a boundary where:

- **Change rates differ** — separate what changes weekly from what changes yearly.
- **The reason to change differs** — different stakeholders, different domains.
- **A dependency should be inverted** — anything volatile the domain must not know about.
- **A team boundary exists** — ownership and module boundaries should coincide.
- **The vocabulary changes** — where a word starts meaning something different, you are in a
  different context.

Do **not** draw one where technical type differs but purpose does not, where the split is
speculative, or where both sides must always change together anyway.

## How to use it

Get the boundary right **in the monolith first**. A clean internal boundary can be extracted
later; a bad one becomes a distributed bad one.

```go
// RIGHT — a real boundary: different reason to change, inverted dependency,
// vocabulary genuinely differs either side.
internal/policy/     // "change", "verdict", "rule"      — changes with policy rules
internal/matrix/     // "suite", "coverage", "plan"      — changes with test tooling
internal/forge/      // "pull request", "check run"      — changes with the provider's API

// policy declares what it needs; matrix and forge do not appear in its imports.
```

Split into a separate **process** only for a reason genuinely about deployment or runtime:
independent deployment cadence that demonstrably matters, a genuinely different scaling
profile, isolation of failure or of a trust boundary, a different technology genuinely
required, or team autonomy at a scale where coordination cost is real.

**The invariant: "it's cleaner" is not a reason to cross a process boundary.** Neither is
"it's the modern approach".

Note the specific cost independent deployability buys: **services that deploy independently
are not tested together.** That is the trade — release autonomy in exchange for losing the
guarantee that the combination works. Make it deliberately.

## How not to use it

**Using a network call to fix a design problem.**

```go
// WRONG — the two packages were tangled, so they became two services.
// Before: a compile error when the contract broke.
verdict := policy.Evaluate(change)

// After: a runtime failure, plus latency, retries, versioning and a new
// deployment to coordinate. The tangle is unchanged — it is now distributed.
verdict, err := policyClient.Evaluate(ctx, &pb.EvaluateRequest{Change: toProto(change)})
// Symptom: the same two things still change together, and now a release needs both.
```

**A boundary between two things that always change together.**

```go
// WRONG — a port between packages that are edited in the same commit every time.
package change
type Builder interface{ Build(Event) (Document, error) }   // one implementation, ever

package changeimpl
type builder struct{}
// Symptom: every change to the document shape edits the interface, the
// implementation and every fake. That is one module with extra ceremony.
```

**Cyclic dependencies between modules.** Two modules that import each other are one module
pretending to be two. Go will refuse to compile it, which is a kindness — in languages that
permit it, the cycle persists and the boundary is fiction.

**Letting the module boundary and the ownership boundary diverge.** When two teams change one
module, every change needs coordination that the structure does not acknowledge. That is
developmental coupling — see `coupling.md`.

**Trade-off:** a boundary costs translation, an interface, and sometimes a performance hop,
and it is harder to move than the code either side of it.

**Related:** `coupling.md`, `cohesion.md`, `deployability.md`
