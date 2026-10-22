# Errors and assertions

*Construction · Derived from the global `cc2-defensive-design` skill — **binding on this project***

## What it is

Two distinct mechanisms for two distinct causes. **Assertions** are for conditions that must
never occur — a bug in our code. **Error handling** is for conditions expected in production —
the world being the world. The distinction is not about severity; it is about **whose fault it
is**. Alongside: a choice per subsystem between **correctness** (never return a wrong answer;
fail instead) and **robustness** (keep running with a degraded but safe result).

## Why it exists

Because conflating them produces two specific failures. An assertion on user input crashes
the process for something a user can trigger at will — and vanishes entirely in builds where
assertions are disabled. An error path for an impossible state adds handling code for a case
that, if it ever runs, means the program is already wrong and should stop.

The correctness/robustness choice exists because the two **genuinely conflict**. Picking one
per subsystem is a decision; mixing them arbitrarily is an accident.

## When to use which

| | Assertion | Error handling |
|---|---|---|
| Handles | Conditions that must never occur | Conditions expected in production |
| Cause | A bug in our code | The world being the world |
| Examples | Invariant broken, impossible state | Bad input, timeout, missing file, malformed payload |
| Behaviour | Fail loudly and immediately | Degrade deliberately, per policy |
| In production | May be compiled out; never carries side effects | Always active |

**Correctness** is the default for money, auth, anything persisted, and any security decision.
**Robustness** is the default for rendering, telemetry and best-effort background work.

## How to use it

```go
// RIGHT — an expected condition. Wrapped, with context, actionable.
func (i Ingest) fetch(ctx context.Context, pr PullRequest) ([]ChangedFile, error) {
    files, err := i.src.FilesChanged(ctx, pr)
    if err != nil {
        return nil, fmt.Errorf("fetch changed files for %s#%d: %w", pr.Repository, pr.Number, err)
    }
    return files, nil
}
```

```go
// RIGHT — an invariant. Unreachable unless the code is wrong, so it stops.
func effectFor(reasons []Reason) Effect {
    switch {
    case len(reasons) == 0:
        return Admit
    case len(reasons) > 0:
        return Refuse
    }
    panic("unreachable: len is neither zero nor positive")
}
```

```go
// RIGHT — the two postures, chosen per path and stated.
// Security: correctness. Any failure to decide refuses.
func (g Gate) GateSecurity(ctx context.Context, c Change) (Verdict, error) {
    v, err := g.security.Evaluate(ctx, c)
    if err != nil {
        return Verdict{Effect: Refuse, Reasons: []Reason{reasonFor(err)}}, err
    }
    return v, nil
}

// Test selection: robustness. Any failure runs everything, and logs loudly.
func (g Gate) SelectTests(ctx context.Context, c Change) TestPlan {
    plan, err := g.selection.Evaluate(ctx, c)
    if err != nil {
        g.log.Error("test selection failed, running full matrix", "err", err)
        return TestPlanRunAll()
    }
    return plan
}
```

**The invariant: a failure is never a plausible value.** Fail fast and loudly at the point of
detection.

## How not to use it

**The silent default.**

```go
// WRONG — a failure becomes data.
func loadCoverage(path string) []Suite {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil                       // "no coverage" and "could not read" now identical
    }
    return parse(b)
}
// Symptom: test selection runs zero suites because the file was missing, and
// reports success. The error surfaces as a wrong answer, far from its cause.
```

**The empty catch.**

```go
// WRONG — discards the only evidence of the fault.
if err := recorder.Save(ctx, v); err != nil {
    // nothing
}
// Symptom: decisions silently stop being recorded and nothing indicates when it
// started. If a failure is genuinely ignorable, say why — that is the whole
// justification, and it is a legitimate constraint comment.
```

**An assertion on something a user can trigger.**

```go
// WRONG — crashes the process on malformed input, and disappears in a build
// with assertions off.
func parseRepository(s string) Repository {
    parts := strings.Split(s, "/")
    if len(parts) != 2 { panic("bad repository") }    // user-supplied
    return Repository{parts[0], parts[1]}
}
// Symptom: a denial of service reachable by one bad request.
```

**An assertion with a side effect.**

```go
// WRONG — the work vanishes wherever assertions are disabled.
assertTrue(deleteExpiredGrants() > 0)
// Symptom: grants stop being deleted in production only. Assertions read state;
// they never change it.
```

**An unmatched lookup falling through to an undefined value.** Handle the miss explicitly — a
stated default or a deliberate error. See `table-driven-methods.md`.

**Trade-off:** correctness means outages where robustness would have degraded, and robustness
means wrong answers where correctness would have stopped. There is no setting that avoids
both — which is why it is chosen per subsystem and recorded.

**Related:** `defensive-design.md`, `../security/error-handling-and-logging.md`,
`control-flow.md`
