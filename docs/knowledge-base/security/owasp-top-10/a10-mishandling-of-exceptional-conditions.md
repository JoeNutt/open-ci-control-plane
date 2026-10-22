# A10:2025 — Mishandling of Exceptional Conditions

*OWASP Top 10:2025 · External: OWASP*

## What it is

**New in 2025**, spanning roughly 24 weakness types: code that **fails open**, swallows errors
silently, or leaks internal detail when something goes wrong. Error paths treated as a
vulnerability class in their own right rather than as a robustness concern.

Note for anyone carrying 2021 habits: it took the tenth slot SSRF vacated, but it is **not a
renamed SSRF** — SSRF moved into A01. These are unrelated categories.

## Why it is new

Because error paths are the least tested code in most systems and the most reliably exploitable
when they are wrong. A control that works on the happy path and fails open on the error path
is worse than no control, because it is believed.

It is also the category most relevant to this project's own design. The fail-closed /
fail-open asymmetry, the three-way failure split, and the rule that undefined is not false are
all answers to this category — and getting any of them wrong produces a silent admit.

## When to check for it

Every error path in security-relevant code. The questions:

- Does an error on an authorisation or validation path **deny**?
- Can an exception route **around** a check — including via a retry, a background job, or a
  cached result?
- Does a failure return a plausible value rather than an error? `nil`, `0`, `[]`, `false`.
- Is a third state possible — not just success and failure, but **undefined**? For a policy
  engine an empty result is not a denial.
- Does a failure leave partially applied state behind?
- Does the error returned to a caller differ from the one logged?

## How to use it

Classify the failure, choose a direction deliberately per path, and state it in the code.

```go
// RIGHT — three distinct outcomes, each handled. Undefined is not false.
verdict, err := e.Evaluate(ctx, change)
switch {
case errors.Is(err, ErrPolicyUndefined):
    return Verdict{Effect: Refuse, Reasons: []Reason{reasonNoDecision}}, err
case err != nil:
    return Verdict{Effect: Refuse, Reasons: []Reason{reasonEngineError}}, err
default:
    return verdict, nil
}
```

```go
// RIGHT — the asymmetry is stated in the code, not in a comment.
// Security: any failure to decide refuses.
func (g Gate) GateSecurity(ctx context.Context, c Change) (Verdict, error) {
    v, err := g.security.Evaluate(ctx, c)
    if err != nil {
        return Verdict{Effect: Refuse}, err
    }
    return v, nil
}

// Test selection: any failure runs everything — and says so loudly.
func (g Gate) SelectTests(ctx context.Context, c Change) TestPlan {
    plan, err := g.selection.Evaluate(ctx, c)
    if err != nil {
        g.log.Error("selection failed, running full matrix", "err", err)
        return TestPlanRunAll()
    }
    return plan
}
```

**The invariant: a failure is never a plausible value, and the failure direction is a decision
recorded per path.**

## How not to use it

**Failing open on a check.**

```go
// WRONG
allowed, err := policy.MayRead(ctx, caller, id)
if err != nil {
    log.Warn("policy unavailable, allowing", "err", err)
    allowed = true
}
// Symptom: taking the policy store offline grants universal access.
```

**A silent default.**

```go
// WRONG — "could not read" and "nothing to run" become the same answer.
func loadCoverage(path string) []Suite {
    b, err := os.ReadFile(path)
    if err != nil { return nil }
    return parse(b)
}
// Symptom: zero suites run and the build reports success.
```

**Treating an empty result as a negative.**

```go
// WRONG — for a policy engine, an empty result set means UNDEFINED.
rs, _ := query.Eval(ctx, rego.EvalInput(in))
if len(rs) == 0 {
    return Verdict{Effect: Admit}        // "no violations found"
}
// Symptom: a mistyped input field evaluates to undefined forever, and every
// change is admitted. The policy never ran.
```

**Leaking internals on failure.**

```go
// WRONG
http.Error(w, err.Error(), 500)
// Output: pq: relation "decisions_v2" does not exist
// Symptom: the schema is public. Two audiences: the log gets the detail.
```

**A partially applied state left behind.** An operation that half-succeeded — grant recorded,
expiry not set — is frequently exploitable and always confusing.

**Trade-off:** failing closed converts some outages-of-a-dependency into outages-of-the-
product, and the fail-open paths need loud signalling to avoid degrading silently forever.
There is no setting that avoids both, which is why it is chosen and recorded per path.

**Related:** `../error-handling-and-logging.md`, `../../engineering/errors-and-assertions.md`,
`a09-logging-and-alerting-failures.md`
