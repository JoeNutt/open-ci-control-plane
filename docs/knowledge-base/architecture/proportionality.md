# Proportionality

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

Scaling structure to what the system actually is. Full layering on a small tool is waste, and
the ceremony makes it *harder* to change rather than easier. Three rough tiers:

- **Script or small tool:** no layers. Keep I/O separable from logic and stop.
- **Service with real business rules:** ports and adapters. Domain isolated, infrastructure
  behind interfaces.
- **Large or long-lived system:** explicit layers, enforced dependency direction, documented
  boundaries.

## Why it exists

Because the failure mode of architectural advice is over-application. Every principle in this
directory is correct and every one of them costs code, indirection and comprehension. Applied
to a system that does not need them, the costs are paid and the benefits are not — and the
result is genuinely worse than the naive version.

It exists as an explicit principle because "we followed best practice" is a defence people
reach for, and it is not one. A design nobody can trace has failed regardless of which
patterns it contains.

## When to use it

Before adopting any structure, and the test is specific: **are the business rules complex
enough to be worth protecting?** If the system is a thin shell over CRUD, say so rather than
building four layers to move a field from a form to a table.

Also when reviewing: a pass-through layer, a port with one permanent implementation, or an
interface with no second implementor in sight are all proportionality failures.

## How to use it

Match the structure to the tier, and say which tier you judged it to be.

```go
// RIGHT — a small tool. One file, no layers. I/O separated from the logic,
// which is all the structure this earns.
func main() {
    data, err := os.ReadFile(os.Args[1])
    if err != nil { log.Fatal(err) }
    fmt.Println(summarise(parse(data)))     // pure, testable, no seams needed
}
```

```go
// RIGHT — a service with real rules. Ports where volatility crosses, nowhere else.
type Gate struct {
    src   ChangeSource     // port: a forge, volatile, must be swappable
    clock Clock            // port: non-deterministic
}
// No port for the bounds checker. It is pure, stable, and lives in the same module.
```

**The invariant: every interface has a reason that exists today.** "We might need a second
implementation" is not one — `incremental-architecture.md` covers deferring instead.

## How not to use it

**Four layers over a CRUD shell.**

```go
// WRONG — a handler, a service, a repository and a mapper to move one field.
func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
    h.service.Update(toDTO(decode(r)))        // service calls repo, repo calls mapper,
}                                              // mapper returns the same shape back
// Symptom: four files changed to add a field, none of which makes a decision.
// Every layer is pass-through. That is ceremony, not isolation.
```

**A port with one implementation and no prospect of another.**

```go
// WRONG — an interface invented for symmetry.
type BoundsChecker interface{ Check(Change, Limits) (Bounded, error) }

type boundsChecker struct{}                   // the only implementation, forever
// Symptom: indirection with nothing behind it. The concrete function was already
// testable — it is pure. Call it directly.
```

**Microservices as the definition of modularity.** Treating a process split as what modularity
*means* skips the actual work, which is taking the boundaries and the protocols across them
seriously. See `module-boundaries.md`.

**Refusing structure in a system that has outgrown the tier.** The principle cuts both ways: a
long-lived service with real rules and no enforced dependency direction will acquire outward
dependencies silently, and retrofitting is far more expensive than starting with them.

**Trade-off:** judging the tier is a judgement, and getting it wrong in either direction is
costly. The asymmetry worth knowing: too little structure is usually cheaper to fix than too
much, because adding a boundary is easier than removing one code depends on.

**Related:** `incremental-architecture.md`, `module-boundaries.md`, `coupling.md`
