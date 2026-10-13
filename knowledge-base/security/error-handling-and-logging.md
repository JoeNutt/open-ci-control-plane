# Error handling and logging

*Security · Derived from the global `sec-review-testing` skill — **binding on this project***

## What it is

Error paths and logs as security controls. **Fail closed:** an error in an authorisation or
validation path must deny. **Two audiences, two messages:** the caller gets an actionable,
generic message; the log gets full diagnostic context. **Log for detection:** record the
events that reveal an attack, in a form that cannot be forged, and alert on them.

## Why it exists

Both are now OWASP Top 10 categories in their own right — **mishandling of exceptional
conditions**, and **security logging and alerting failures** — and both are easy to leave out
of a threat model because neither looks like a vulnerability while you are writing it.

An error path that routes around a check is a bypass with no evidence. And an undetected
breach is an unbounded one: a log nobody reads and nothing alerts on is storage, not
detection.

## When to use it

Every error path in security-relevant code, and every event worth detecting:

**Log** — authentication attempts and outcomes, authorisation **denials**, privilege changes,
access to sensitive data, administrative actions, input-validation rejections, configuration
changes. Include who, what, when, where from, and the outcome.

**Never log** — passwords, tokens, keys, session ids, full card numbers, or personal data
beyond what is needed.

**Alert** on repeated authorisation denials, credential-stuffing patterns, unusual data
volumes.

## How to use it

```go
// RIGHT — fails closed, logs the diagnostic, returns the generic.
func (h Handler) GetDecision(w http.ResponseWriter, r *http.Request) {
    caller := h.caller(r)
    d, err := h.repo.DecisionForCaller(r.Context(), id, caller)
    if err != nil {
        h.log.Warn("decision lookup denied or failed",
            "caller", caller.ID, "decision", id, "remote", r.RemoteAddr, "err", err)
        http.Error(w, "not found", http.StatusNotFound)   // no detail to the caller
        return
    }
    writeJSON(w, d)
}
```

```go
// RIGHT — structured fields, so an untrusted value cannot forge an entry.
h.log.Info("authorization denied",
    "caller", caller.ID,
    "object", id,
    "reason", reason)       // a newline in `reason` is a field value, not a new line
```

```go
// RIGHT — the fail-open path still fails loud.
plan, err := g.selection.Evaluate(ctx, c)
if err != nil {
    g.log.Error("test selection failed, running full matrix", "err", err)
    return TestPlanRunAll()      // open on the decision, loud in the log
}
```

**The invariant: an error on a security or validation path denies, and never routes around a
check.** And a redacting type, not care at each call site, is what keeps secrets out — see
`secret-management.md`.

## How not to use it

**An error that proceeds.**

```go
// WRONG — the exception routed around the control.
allowed, err := policy.MayRead(ctx, caller, id)
if err != nil {
    log.Warn("policy unavailable, allowing", "err", err)
    allowed = true                          // fail open on an authorisation check
}
// Symptom: taking the policy store offline grants universal access.
```

**Internal detail returned to the caller.**

```go
// WRONG — leaks the driver, the schema and the query.
http.Error(w, err.Error(), http.StatusInternalServerError)
// Output: pq: relation "decisions_v2" does not exist
// Symptom: an attacker maps the schema from error text. Two audiences: the log
// gets this, the caller gets "internal error".
```

**Errors that enumerate.**

```go
// WRONG — tells an attacker which accounts and objects exist.
if user == nil  { return errors.New("no such user") }
if !pwOK        { return errors.New("wrong password") }
// Symptom: account enumeration. Return one generic failure — and watch response
// timing, which enumerates just as well.
```

**A swallowed error in security-relevant code.**

```go
// WRONG — a silent catch is a bypass with no evidence.
_ = h.audit.Record(ctx, denial)
// Symptom: denials stop being recorded and nothing indicates when it began.
```

**A concatenated log message carrying untrusted data.** Newlines forge entries; an untrusted
value in a format string has produced remote code execution. Structured fields only.

**Leaving a partially applied state behind on failure.** An operation that half-succeeded is
frequently exploitable — and always confusing.

**Trade-off:** generic caller messages make legitimate debugging harder and generate support
load, and a correlation id must be threaded everywhere to make the logs usable. Both are the
price of not handing the attacker a map.

**Related:** `../engineering/errors-and-assertions.md`, `secret-management.md`,
`security-review-and-testing.md`
