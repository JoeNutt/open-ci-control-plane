# A06:2025 — Insecure Design

*OWASP Top 10:2025 · External: OWASP*

## What it is

A flaw in what the system was designed to do, rather than a defect in how it was built. A
missing control that was never specified, a workflow that can be completed out of order, a
business rule enforced only on the client, a recovery flow weaker than the login it protects,
a trust assumption that does not hold. **Down from #4 to #6.**

## Why it exists as a separate category

Because **a perfect implementation of an insecure design is still insecure**, and no amount of
code review, static analysis or input validation finds it. The defect is the absence of a
requirement, and tooling cannot detect an absence it was never told to expect.

The distinction is the actionable part: a bug is fixed in code, and a design flaw is fixed by
changing what the system does. Discovering one late means the fix is a feature change.

## When to check for it

Before implementation, by threat modelling — which exists precisely for this category. The
questions:

- What is the **attacker's goal** here, and what would they have to do? Model the misuse
  case, not just the use case.
- Can any step be skipped, repeated, or done out of order?
- Is any limit — rate, quantity, value, retry — enforced where the client cannot reach it?
- Is there a **second control** if this one fails? If not, this one is load-bearing and needs
  to be excellent.
- Does any flow grant more than the one it bypasses — password reset, support impersonation,
  break-glass?

## How to use it

The standard is `../threat-modeling.md`. Here, the design shapes that resist the category:
make illegal states unrepresentable, and make limits structural.

```go
// RIGHT — the workflow cannot be completed out of order, because the types
// forbid it. There is no function that accepts Raw and dispatches.
type Raw struct{ files []ChangedFile }
type Bounded struct{ files []ChangedFile }
type Decided struct{ verdict Verdict }

func CheckBounds(Raw, Limits) (Bounded, error)
func Evaluate(Bounded) Decided
func Dispatch(Decided) error              // cannot be reached with Raw
```

```go
// RIGHT — break-glass bounded by design: separate credential, hard ceiling,
// mandatory attribution, cannot be extended.
type Grant struct {
    actor     string        // mandatory
    reason    string        // mandatory, 20-500 chars
    expiresAt time.Time     // max 1h, never extendable
}
// At most 5 live grants per installation. Break-glass that is always available
// is not break-glass.
```

**The invariant: every control has a named second line, or is documented as load-bearing.**

## How not to use it

**A limit the client enforces.**

```go
// WRONG — the cap is in the request.
plan := selectSuites(change, req.MaxSuites)
// Symptom: the caller sends MaxSuites: 0 and nothing runs. Or 100000 and the
// runner is exhausted. The attacker does not use your client.
```

**A workflow with skippable steps.**

```go
// WRONG — three functions over one mutable struct, order enforced by convention.
c := &Change{}
buildDocument(c)
checkBounds(c)      // if omitted, nothing fails
evaluate(c)
// Symptom: a new caller omits checkBounds and the compiler is silent.
```

**A recovery path stronger than the thing it recovers.**

```go
// WRONG — break-glass on the enforcement path's own credential.
mux.Handle("/v1/breakglass-grants", grantHandler)     // same listener, same auth
// Symptom: compromising the webhook path yields the power to exempt any
// repository. The bypass is easier than the thing bypassed.
```

**Designing only the use case.** If the model has no misuse case, nobody asked what an
attacker wants.

**Trade-off:** threat modelling costs time before any code exists and produces a list of
things that have not happened — which is also the only point at which these fixes are still
cheap.

**Related:** `../threat-modeling.md`, `a01-broken-access-control.md`,
`../../architecture/proportionality.md`
