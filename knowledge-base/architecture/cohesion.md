# Cohesion

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

The degree to which everything in a module contributes to one well-defined job. The strongest
form is **functional** cohesion — everything present serves one purpose. The weakest is
**coincidental** — things grouped because they are the same *kind* of thing rather than
because they serve the same purpose.

## Why it exists

It is the counterpart to coupling, and the two trade off: pursuing low coupling by splitting
aggressively produces modules with no coherent purpose, which is its own cost. Cohesion is
what stops "reduce coupling" from degenerating into "scatter everything".

It is also the deeper reason to organise by feature rather than by technical type. `handlers/`,
`services/`, `models/` is maximally *coincidental* cohesion — it groups by what things are,
guaranteeing that every real change touches three directories.

## When to use it

Apply the test whenever deciding what goes in a module, and whenever a module feels too large:

**Do the things in this module change together, for the same reason, at the same time?**

If half a module changes on every release and the other half has not changed in two years, it
is two modules. The same signal at type level: if a type's fields split into groups used by
different methods, it is already two types.

## How to use it

Group by the thing that changes together, and let the technical kind fall where it may.

```go
// RIGHT — one package, one job. Everything here changes when the bounds rules change.
package bounds

type Limits struct{ MaxFiles, MaxPatchBytes, MaxDepth int }

func Check(c Change, l Limits) (Bounded, error) { /* ... */ }
func (l Limits) Describe() string               { /* ... */ }
func parseLimits(cfg []byte) (Limits, error)    { /* ... */ }
```

```go
// RIGHT — a type whose fields are all used by all its methods.
type Grant struct {
    id        GrantID
    expiresAt time.Time
}

func (g Grant) IsLive(now time.Time) bool { return now.Before(g.expiresAt) }
func (g Grant) ID() GrantID               { return g.id }
```

**The invariant: one module, one reason to change.** If you can name two reasons, name two
modules.

## How not to use it

**Grouping by technical kind.**

```go
// WRONG — package shapes the code by what things ARE, not what they are FOR.
package models   // Change, Verdict, Grant, Coverage, TelemetrySnapshot
package services // GateService, MatrixService, GrantService, TelemetryService
package handlers // GateHandler, MatrixHandler, GrantHandler
// Symptom: adding one field to Grant touches three packages, and nothing stops
// MatrixService reaching into Grant's internals because they are already neighbours.
```

**A type whose fields serve different actors.**

```go
// WRONG — two responsibilities wearing one name.
type Engine struct {
    query      *rego.PreparedEvalQuery   // used by Evaluate
    bundleETag string                    // used by Refresh
    httpClient *http.Client              // used by Refresh
    metrics    Recorder                  // used by Evaluate
}
// Symptom: the field groups do not overlap. Evaluate and Refresh share nothing.
// That is an Evaluator and a BundleLoader.
```

**A `shared`, `common` or `utils` package.** Zero cohesion by construction — the only thing
its contents have in common is that nobody knew where else to put them. It then acquires
dependencies in every direction and becomes a second coupling graph under the intended one.
The global standards ban the names outright; split by concern instead.

**Splitting until nothing has a purpose.** A package per type, or a file per function, is low
coupling bought with no cohesion. The modules are individually trivial and the system is
unreadable as a whole.

**Trade-off:** feature-first organisation means the same *kind* of thing lives in several
places, which is mildly worse for someone looking for "all the HTTP handlers" and much better
for everyone making a change.

**Related:** `coupling.md`, `layering.md`, `module-boundaries.md`
