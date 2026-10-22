# Classes and abstractions

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

A type models a thing in the problem domain, and its interface reads as operations on that
thing. **Size is measured in responsibilities, not fields.** Cohesion is what binds code
answering to a *single actor* — one source of change requests. Default to the most private
visibility available; prefer containment over inheritance; hold one level of abstraction per
interface.

## Why it exists

Because the question "is this class too big?" has no useful answer in lines or fields. The
actor test does: if `calculatePay()` answers to finance, `reportHours()` to HR and `Save()`
to the DBAs, that is three types wearing one name, and a change requested by one will break
another.

Information hiding exists because a module that conceals nothing protects callers from
nothing. Ask of every type: **what does this hide?** If the answer is "nothing", it is a data
bag with methods.

## When to use it

Designing a new type, and whenever an existing one has grown. The mechanical smell for the
same thing: **if the methods split into groups that each use a different subset of the
fields, the type is already two types.**

Also when reaching for inheritance. Use it only for genuine "is a" substitutability, at most
three levels deep. Anything a subclass must override to neuter means the relationship is
containment.

## How to use it

```go
// RIGHT — one actor (policy owners), one secret (how a verdict is computed),
// every field used by every method.
type Gate struct {
    rules   []Rule
    failOpen bool
}

func (g Gate) Decide(c Change) Verdict   { /* ... */ }
func (g Gate) Explain(c Change) []Reason { /* ... */ }
```

```go
// RIGHT — containment, not inheritance. The capability is composed in.
type AuditedGate struct {
    gate     Gate              // has-a
    recorder DecisionRecorder
}
```

```go
// RIGHT — the smallest visible surface. Only Decide is exported.
func (g Gate) Decide(c Change) Verdict { return g.combine(g.match(c)) }
func (g Gate) match(c Change) []Rule   { /* unexported */ }
func (g Gate) combine(r []Rule) Verdict { /* unexported */ }
```

**The invariant: one type, one actor.** Name who asks for changes to it. If you can name two,
name two types.

## How not to use it

**A type serving several actors.**

```go
// WRONG — three actors, one name.
type Engine struct {
    query      *rego.PreparedEvalQuery
    bundleETag string
    httpClient *http.Client
    metrics    Recorder
}
func (e *Engine) Evaluate(...)  { /* uses query, metrics */ }
func (e *Engine) Refresh(...)   { /* uses bundleETag, httpClient */ }
func (e *Engine) MinutesSaved() { /* uses metrics */ }
// Symptom: the field groups do not overlap. That is an Evaluator, a BundleLoader
// and a Telemetry reporter.
```

**A data bag that hides nothing.**

```go
// WRONG — fields exported, no behaviour, no secret.
type Decision struct {
    Effect  string
    Reasons []string
    Raw     map[string]any     // the internal representation, public
}
// Symptom: every caller can construct an invalid Decision, and changing Raw's
// shape is a breaking change to everyone. Nothing is concealed.
```

**Reaching through a chain.**

```go
// WRONG — binds the caller to a structure two types away.
sha := event.PullRequest().Head().Commit().SHA()
// Symptom: a change inside Head() breaks this call site, which never names Head's
// type. Talk to immediate collaborators; ask for what you need.
```

**Exporting a routine because it happens to call other exported routines.** Visibility is a
decision about the abstraction, not a consequence of the call graph.

**Mixing abstraction levels in one interface.** A public surface carrying both `ApproveChange`
and `SetHeaderBytes` makes the caller operate at two altitudes — see `code-layout.md`.

**Trade-off:** splitting by actor means more types and more wiring, and a change that genuinely
spans actors now touches several files. That is the honest cost of the change for one actor not
breaking another.

**Related:** `routines.md`, `../architecture/cohesion.md`, `code-layout.md`
