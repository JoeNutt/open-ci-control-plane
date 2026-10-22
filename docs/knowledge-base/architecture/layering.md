# Layering — what goes where

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

A procedure for deciding which layer a piece of code belongs to, and what each layer is
allowed to contain. Four questions, asked in order; the first "yes" is the answer.

1. **Would this rule still exist if the product were a paper form?** → domain.
2. **Does it orchestrate a specific thing a user or system does?** → use case.
3. **Does it translate between the domain and something outside?** → adapter.
4. **Is it a specific technology?** → framework or driver, wired in `main`.

If a piece of code answers yes to two, it is doing two jobs. Split it.

## Why it exists

"Where should this go?" is the question that gets answered by habit, and habit puts everything
in whichever file is already open. The result is business rules in HTTP handlers, SQL in
domain types, and a codebase where the layers exist in the folder names only.

The procedure exists because the answer is not obvious from the code's *subject* — it follows
from what the code is *for*. Two functions both about pull requests can belong in different
layers.

## When to use it

Whenever adding a file and the destination is not already obvious, and whenever reviewing
code that feels like it is in the wrong place. The four questions are also the fastest way to
show that something is doing two jobs — it answers yes twice.

## How to use it

Organise **by feature, then by layer**, so things that change together sit together.

```
internal/
  policy/                      # feature
    domain/                    # entities, invariants, ports
    application/              # use cases
    adapters/
      forge/                   # driving + driven adapters
      bundle/
  matrix/
    domain/ application/ adapters/
cmd/controlplane/main.go       # composition root
```

```go
// RIGHT — domain. A business truth, no I/O, no clock, no framework.
package domain

type Verdict struct{ Effect Effect; Reasons []string }

func (c Change) TouchesAuthWithoutTest() bool {
    return c.Kinds.Has(KindAuthz) && !c.Kinds.Has(KindTest)
}
```

```go
// RIGHT — use case. Orchestrates; holds no business truth of its own.
// Declares the ports it needs. Owns the transaction and the authorisation check.
package application

type RunGate struct {
    src    domain.ChangeSource
    record domain.DecisionRecorder
}

func (u RunGate) Execute(ctx context.Context, pr domain.PullRequest) (domain.Verdict, error) {
    change, err := u.src.FilesChanged(ctx, pr)
    if err != nil {
        return domain.Verdict{}, err
    }
    v := domain.Evaluate(change)
    return v, u.record.Save(ctx, pr, v)
}
```

**The invariant: the dependency direction is enforced by a tool, not by a README.** A lint
rule, an architecture test, or a package-boundary check. A convention documented and enforced
by nothing will be violated silently, usually within weeks.

```go
// RIGHT — an architecture test is cheap and catches the regression.
func TestDomainImportsNothingOutward(t *testing.T) {
    for _, imp := range importsOf(t, "./internal/policy/domain/...") {
        if isInfrastructure(imp) {
            t.Errorf("domain imports %s", imp)
        }
    }
}
```

## How not to use it

**Organising by layer first.**

```
controllers/   OrderController, BillingController, GateController
services/      OrderService, BillingService, GateService
models/        Order, Invoice, Change
```

```go
// Symptom: every real change touches three directories, and it becomes invisible
// when `billing` starts reaching into `orders`' internals. This is the layout that
// produces coincidental cohesion — see cohesion.md.
```

**An anemic domain: data holders plus "service" procedures.**

```go
// WRONG — the domain type has no behaviour, so the rules live outside it.
package domain
type Change struct { Kinds []string }      // fields and nothing else

package application
func (s GateService) Execute(c domain.Change) Verdict {
    if contains(c.Kinds, "authz") && !contains(c.Kinds, "test") {   // domain rule, wrong layer
        return Verdict{Effect: Block}
    }
    return Verdict{Effect: Allow}
}
// Symptom: the business rule cannot be found from the type it is about, and the
// same rule gets reimplemented in the next use case that needs it.
```

**Putting the transaction in the repository.** The repository cannot see the whole operation;
the use case can. Likewise the authorisation *decision* belongs where the actor and the object
are both known — the use case or the domain — even if enforcement is also at the edge.

**DTOs within a layer.** Mapping between boundaries is the point and the duplication between a
database row, a domain entity and an API response is deliberate — they change for different
reasons. Mapping *inside* one layer is pass-through ceremony.

**Trade-off:** the mapping tax. Three representations of nearly the same thing, maintained
separately. `dry-across-boundaries.md` explains why that is correct rather than accidental.

**Related:** `dependency-direction.md`, `ports-and-adapters.md`, `cohesion.md`
