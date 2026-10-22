# Defensive design — the barricade

*Construction · Derived from the global `cc2-defensive-design` skill — **binding on this project***

## What it is

A thin boundary layer whose entire job is to convert untrusted input into trusted, well-formed
domain data and reject everything else. Outside the barricade: users, network, files,
environment, databases, other services, other teams' modules — all hostile until proven
otherwise. Inside: data already checked, which interior code may trust without re-checking.

## Why it exists

Because the alternative — check everything everywhere — is how validation logic metastasises
through a codebase. Scattered re-validation is **not** defence in depth: it is two validators
that will eventually disagree, and the looser one silently defines the real contract while the
stricter one lies about it.

Validating once at a known place also makes "has this been checked?" a question with an
answer. Converting at the same place makes it a question nobody needs to ask, because being
inside is visible in the type.

## When to use it

At every entry point where untrusted data arrives: request handlers, deserialisation of any
external payload, configuration and environment loading at startup, reads from a datastore
your code does not exclusively own, and any public entry point of a library others call.

**Trust boundaries are per-system, not universal.** Data trusted inside service A is untrusted
at service B's edge even if you wrote both. "It was validated upstream" is not a guarantee you
can hold when upstream deploys separately. Your own database is a boundary when anything else
can write to it — including an older version of your own code.

## How to use it

Parse, don't validate: return the **converted** value, never a boolean about a string the
caller still holds.

```go
// RIGHT — the type cannot exist without having passed validation.
type Bounded struct{ files []ChangedFile }    // unexported field, no literal construction

func CheckBounds(raw []ChangedFile, l Limits) (Bounded, error) {
    if len(raw) > l.MaxFiles {
        return Bounded{}, fmt.Errorf("%w: %d exceeds %d", ErrTooManyFiles, len(raw), l.MaxFiles)
    }
    return Bounded{files: raw}, nil
}

// Interior code takes Bounded and trusts it. No re-checking, no nil guards.
func Evaluate(b Bounded) Verdict { /* ... */ }
```

```go
// RIGHT — the barricade outputs domain types, not strings.
func parseRepository(s string) (Repository, error) {
    owner, name, ok := strings.Cut(s, "/")
    if !ok || owner == "" || name == "" {
        return Repository{}, ErrMalformedRepository
    }
    return Repository{Owner: owner, Name: name}, nil
}
```

**The invariant: validate at the boundary, completely, once — and convert there.** If a
routine cannot tell whether it is inside or outside, say so explicitly in its contract rather
than defending both ways.

## How not to use it

**Defensive checking everywhere.**

```go
// WRONG — the same field re-checked at four depths.
func Evaluate(b Bounded) Verdict {
    if b.files == nil { return Verdict{} }            // already guaranteed
    for _, f := range b.files {
        if f.Path == "" { continue }                  // already guaranteed
        if len(f.Path) > 4096 { continue }            // a second, looser limit
    }
}
// Symptom: noise, and two places disagreeing about the rules. The 4096 here and
// the MaxDepth at the boundary are now both "the limit".
```

**Passing raw strings inward.**

```go
// WRONG — the boundary validated and then handed the string on anyway.
if !isValidRepository(s) { return ErrMalformed }
return gate.Decide(ctx, s)                 // interior still holds a string
// Symptom: every interior function must wonder whether it was checked. The
// evidence of the check was discarded — see parse, don't validate.
```

**The barricade too deep.**

```go
// WRONG — the handler passes the raw request straight in.
func (h Handler) Post(w http.ResponseWriter, r *http.Request) {
    h.gate.Decide(r.Context(), r)          // validation happens somewhere inside
}
// Symptom: every routine the request passed through was implicitly a boundary
// routine without being written as one.
```

**Validation duplicated in two layers with different rules.** Two barricades that disagree are
worse than one. Extend the existing boundary rather than adding a second.

**Trade-off:** domain types at the boundary mean conversion code and more types — and
occasionally an awkward moment where interior code genuinely wants the raw form. That is
usually the signal that the domain type is missing a method, not that the raw form should be
carried.

**Related:** `errors-and-assertions.md`, `../security/input-validation.md`,
`../architecture/layering.md`
