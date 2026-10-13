# Null Object

*Behavioral · **not GoF** · Authored for this project*

## What it is

An implementation of an interface that does nothing, neutrally, and is supplied in place of
absence. Callers hold a real object and call it unconditionally, so the null check disappears
from every call site into a single decision about which implementation to inject.

Catalogues split on whether this is behavioral or structural; either is defensible. Grouped
behavioral here because it substitutes behaviour.

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
type Observer interface { Record(d time.Duration, err error) }

type discardObserver struct{}                      // named for its behaviour
func (discardObserver) Record(time.Duration, error) {}

// Chosen once, at wiring. Callers never branch.
func NewEngine(obs Observer) *Engine {
    if obs == nil { obs = discardObserver{} }
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
- **Where the caller needs to know.** If "no handler configured" is a condition someone must
  act on, swallowing it removes the only evidence. Make absence an error at startup instead.
- **Returning a null object from a query.** A null `User` whose name is `""` propagates a
  fiction into business logic, and the error surfaces far from the lookup that failed. An
  absent record is better returned as an explicit absence the caller must handle.
- **A null object that returns a plausible value** rather than a neutral one — zero, empty
  string, empty list — where the caller cannot distinguish "nothing happened" from "the
  answer is zero".
- **Hiding a wiring bug.** If a collaborator should always be present, defaulting to a null
  object means a misconfiguration runs silently in production instead of failing at startup.
  Require it; do not default it.

**Trade-off:** it removes a visible check and with it the visible evidence that absence was
possible at all — so a genuinely missing dependency becomes a quiet no-op rather than a
crash.

**Related:** `strategy.md`, `../structural/proxy.md`
