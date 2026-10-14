# Null Object

*Behavioral · **not GoF** · Authored for this project*

## What it is

An implementation of an interface that does nothing, neutrally, and is supplied in place of
absence. Callers hold a real object and call it unconditionally, so the null check disappears
from every call site into a single decision about which implementation to inject.

Catalogues split on whether this is behavioral or structural; either is defensible. Grouped
behavioral here because it substitutes behaviour.

## Why it exists

An optional collaborator is usually represented by its absence, so every call site must check
for nil before calling it. Each check is a chance to forget, and a forgotten one is a nil
dereference in production. The null object turns "absent" into an implementation that does
nothing, so the decision is made once, at wiring, and call sites call unconditionally.

## When to use it

- The **same null check is repeated across many call sites** for one collaborator, and doing
  nothing is the correct behaviour when it is absent.
- The collaborator is genuinely optional: a metrics recorder, a tracer, an audit sink, a
  progress reporter.
- Absence is not an error and never needs reporting.

## How to use it

Make it explicit, name it for what it does, and inject it at the composition root so no
caller ever sees a nullable.

```go
// RIGHT — named for its behaviour, and chosen once at wiring.
type Observer interface {
	Record(d time.Duration, err error)
}

type discardObserver struct{}

func (discardObserver) Record(time.Duration, error) {}

func NewEngine(obs Observer) *Engine {
	if obs == nil {
		obs = discardObserver{}
	}
	return &Engine{obs: obs}
}
```

The invariant: **doing nothing is a correct outcome for every method on the interface.** If
any method must return a meaningful value, this pattern does not apply to that interface.

## How not to use it

- **For anything on a failure path.** A null logger that discards an error, a null validator
  that approves everything, a null authoriser that permits — each converts a missing
  collaborator into a silent wrong answer. This is the serious misuse: absence of a security
  control must fail closed, loudly, not neutrally.

```go
// WRONG — the stand-in for a missing authoriser permits.
type allowAll struct{}

func (allowAll) MayMerge(Caller, PullRequest) bool { return true }
// Symptom: when the permissions service is not configured, every caller may merge.
```

- **Where the caller needs to know.** If "no handler configured" is a condition someone must
  act on, swallowing it removes the only evidence. Make absence an error instead.

```go
// WRONG — an unmatched event kind is answered with a handler that drops it.
func (r Router) For(kind EventKind) Handler {
	if h, ok := r.handlers[kind]; ok {
		return h
	}
	return discardHandler{}
}
// Symptom: a new event kind from the forge is acknowledged and dropped, and nothing records it.
```

- **Returning a null object from a query.** A null `User` whose name is `""` propagates a
  fiction into business logic, and the error surfaces far from the lookup that failed. An
  absent record is better returned as an explicit absence the caller must handle.

```go
// WRONG — a missing record answered with a null user.
func (s Store) User(id UserID) User {
	if u, ok := s.users[id]; ok {
		return u
	}
	return nobody
}
// Symptom: an approval is attributed to an empty name, far from the lookup that missed.
```

- **A null object that returns a plausible value** rather than a neutral one — zero, empty
  string, empty list — where the caller cannot distinguish "nothing happened" from "the
  answer is zero".

```go
// WRONG — the null coverage source answers a real-looking number.
type noCoverage struct{}

func (noCoverage) LinesCovered(path string) int { return 0 }
// Symptom: with no coverage artifact, every file reads as untested — indistinguishable from one no test touches.
```

- **Hiding a wiring bug.** If a collaborator should always be present, defaulting to a null
  object means a misconfiguration runs silently in production instead of failing at startup.
  Require it; do not default it.

```go
// WRONG — a required collaborator defaulted to nothing.
func NewGate(store PolicyStore) *Gate {
	if store == nil {
		store = emptyPolicyStore{}
	}
	return &Gate{store: store}
}
// Symptom: a deployment missing its policy store configuration starts cleanly and evaluates no policies.
```

**Trade-off:** it removes a visible check and with it the visible evidence that absence was
possible at all — so a genuinely missing dependency becomes a quiet no-op rather than a
crash.

**Related:** `strategy.md`, `../structural/proxy.md`
