# A09:2025 — Security Logging and Alerting Failures

*OWASP Top 10:2025 · External: OWASP*

## What it is

Attacks that are not detected because the events were not recorded, not recorded usefully, or
recorded and never acted on. **Stable at #9, and renamed** — 2021's *Logging and Monitoring*
became *Logging and Alerting*, because monitoring that nobody is paged by is not detection.

## Why the rename matters

It moves the bar. Under the old framing a service with structured logs and a dashboard
passed. Under the new one the question is whether anything **fires**: a log nobody reads and
nothing alerts on is storage, not detection.

The category exists because **an undetected breach is an unbounded one.** Every other
category is about preventing an incident; this one is about bounding how long one lasts.

## When to check for it

**Log:** authentication attempts and outcomes, authorisation **denials**, privilege changes,
access to sensitive data, administrative actions, input-validation rejections, configuration
changes. With who, what, when, where from, and the outcome.

**Never log:** passwords, tokens, keys, session ids, full card numbers, or personal data
beyond what is needed.

**Alert on:** repeated authorisation denials, credential-stuffing patterns, unusual data
volumes — and, for this project, **bundle activation failure**, which is the single most
important operational signal in the system.

Then: is there a correlation id through the request path? Are clocks synchronised? Is
retention long enough to investigate, and tamper-resistant?

## How to use it

```go
// RIGHT — structured fields, so an untrusted value is a value and not a new line.
h.log.Warn("authorization denied",
    "caller", caller.ID,
    "tenant", caller.TenantID,
    "object", id,
    "remote", r.RemoteAddr,
    "correlation", corrID,
    "reason", reason)
```

```go
// RIGHT — the fail-open path is loud. Degrading silently is the failure this
// category names.
plan, err := g.selection.Evaluate(ctx, c)
if err != nil {
    g.log.Error("test selection failed, running full matrix", "err", err, "correlation", corrID)
    g.metrics.Inc("selection_fallback_total")        // something can alert on this
    return TestPlanRunAll()
}
```

**The invariant: every security-relevant outcome is recorded, and at least one of them has an
alert attached.** If nothing in the system can page anyone, this category is unaddressed
however good the logs are.

## How not to use it

**Logging only failures of the happy path, not denials.**

```go
// WRONG — the denial is the event worth detecting, and it is not recorded.
if !allowed {
    http.Error(w, "", http.StatusForbidden)
    return
}
// Symptom: an attacker probing a hundred object ids leaves no trace. Repeated
// denials are the clearest signal of enumeration in progress.
```

**A concatenated message.**

```go
// WRONG
log.Printf("denied %s for %s: %s", caller.ID, id, reason)
// Symptom: a newline in `reason` forges an entry. Structured fields cannot be
// escaped out of.
```

**Secrets in the log.**

```go
// WRONG
log.Error("forge call failed", "token", token, "err", err)
// Symptom: the credential is now in every system that ingests logs.
```

**A swallowed error in security-relevant code.**

```go
// WRONG
_ = h.audit.Record(ctx, denial)
// Symptom: audit records stop and nothing indicates when. A silent catch on an
// audit path is a bypass with no evidence.
```

**Logs with no correlation id.** Each line is true and the request cannot be reconstructed,
which is the difference between an investigation and a guess.

**Trade-off:** useful logging means threading a correlation id everywhere, a redaction
discipline at every formatting point, and retention cost. And alerts have a false-positive
budget — too many and they are muted, which returns you to this category by another route.

**Related:** `../error-handling-and-logging.md`, `../secret-management.md`,
`a10-mishandling-of-exceptional-conditions.md`
