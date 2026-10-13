# A01:2025 — Broken Access Control

*OWASP Top 10:2025 · External: OWASP*

## What it is

A request is permitted that the actor was not entitled to make. The largest category in the
2025 edition at roughly **40 mapped CWEs**, spanning object-level failures (reading a record
by changing an id), function-level failures (an admin route reachable by a normal user),
missing tenant isolation, privilege escalation, and — **new for 2025** — **SSRF**, absorbed
on the reasoning that tricking a server into making a request it should not is an
access-control failure.

## Why it exists as the top risk

Because it has been ranked **#1 since 2021 and stayed there**, and because **a missing check
looks exactly like code that works.** Nothing fails, no test goes red, no scanner fires — the
endpoint returns the right answer to the wrong person. It is simultaneously the most exploited
class and the least detectable by tooling.

## When to check for it

Every route, handler, message consumer and background job. The questions:

- Is the **object** checked, or only the action? Every read or write by an id from user input
  is an object-level check to verify.
- Is the tenant predicate enforced **structurally** — a scoped connection, row-level
  security, a repository that requires a tenant — or written per query?
- Is the check reachable from **every** path, including error paths, retries, admin tools and
  step three of a wizard?
- Does a policy-lookup failure deny, or continue?
- Is any URL fetched server-side from caller-influenced input? That is the SSRF sub-case.

## How to use it

The standard is `../authorization.md` and `../least-privilege.md`. Here, the detection shape:
scope the lookup so an unauthorised result cannot exist, and allowlist any outbound
destination.

```go
// RIGHT — the actor is part of the query. There is nothing unauthorised to leak.
const q = `SELECT id, effect FROM decisions WHERE id = $1 AND tenant_id = $2`
row := r.db.QueryRowContext(ctx, q, id, caller.TenantID)
```

```go
// RIGHT — the SSRF sub-case: allowlist the destination, refuse redirects.
if !allowedBundleHosts[u.Host] {
    return ErrHostNotAllowed
}
client := &http.Client{
    Timeout:       30 * time.Second,
    CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}
```

**The invariant: there is a test proving the stranger is refused.** A suite that proves the
owner can read is green with the check deleted.

## How not to use it

**Authenticated, not authorised.**

```go
// WRONG — any logged-in caller reads any decision.
if h.session.User(r) == nil { http.Error(w, "", 401); return }
d, _ := h.repo.Decision(ctx, DecisionID(r.PathValue("id")))
writeJSON(w, d)
// Symptom: incrementing the id in the URL walks every tenant.
```

**The tenant from the request.**

```go
// WRONG — the client asserts its own scope.
tenant := r.Header.Get("X-Tenant-ID")
// Symptom: one header change reads another tenant. A tenant id in a header,
// body, cookie or unsigned claim is input, not fact.
```

**Blocking internal ranges instead of allowlisting — the SSRF failure.**

```go
// WRONG — denylisting loses.
if isPrivateIP(host) { return ErrBlocked }
resp, _ := http.Get(req.BundleURL)
// Symptom: fails to DNS rebinding, redirects, alternate IP encodings and IPv6.
// 169.254.169.254 reached through a redirect returns cloud credentials.
```

**Relying on the UI hiding an action, or an unguessable URL.** Both are client-side; neither
is a control.

**Trade-off:** structural tenant enforcement makes legitimate cross-tenant work — support
tooling, reporting — genuinely awkward, and those paths need their own audited design rather
than a bypass.

**Related:** `../authorization.md`, `../least-privilege.md`, `a06-insecure-design.md`
