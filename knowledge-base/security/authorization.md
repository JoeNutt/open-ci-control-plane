# Authorization

*Security · Derived from the global `sec-authz-least-privilege` skill — **binding on this project***

## What it is

Deciding whether **this actor** may do **this thing** to **this object**, right now. Distinct
from authentication, which asks who the actor is. Deny by default: every action requires an
explicit grant, authorised in one place so a new endpoint is protected by construction rather
than by the author remembering.

## Why it exists

**Broken access control is the most exploited vulnerability class in production software**,
and it is the least likely to be caught by a scanner — because **a missing check looks exactly
like code that works.** Nothing fails, no test goes red, and the endpoint returns the right
answer to the wrong person.

The object-level rule exists because the common check is the wrong one: verifying the caller
is logged in, or holds a role, but never that this particular record is theirs.

## When to use it

Every request, every handler, every message consumer — and on **every** call, not once at
login. A permission verified at login is stale by the next call; roles change and sessions
outlive them.

Also at every elevation boundary: re-authenticate for sensitive actions even within a valid
session, and re-check after any redirect, workflow step or state transition. Multi-step flows
are routinely protected at step one and open at step three.

## How to use it

Scope the query so the object cannot be fetched at all. A filter in the data access layer
cannot be forgotten by a later `if`, and it fails safe when someone adds a new read path.

```go
// RIGHT — the actor is part of the lookup. There is no unauthorised result to leak.
func (r Repo) DecisionForCaller(ctx context.Context, id DecisionID, c Caller) (Decision, error) {
    const q = `SELECT id, effect FROM decisions
               WHERE id = $1 AND tenant_id = $2`
    row := r.db.QueryRowContext(ctx, q, id, c.TenantID)
    // ...
}
```

```go
// RIGHT — deny by default. An unregistered route has no policy and refuses.
func (m Middleware) Wrap(route string, h http.Handler) http.Handler {
    policy, ok := m.policies[route]
    if !ok {
        return denyAll("no policy registered for " + route)   // fails closed
    }
    return policy.Enforce(h)
}
```

```go
// RIGHT — fail closed on a policy error.
allowed, err := m.policy.MayRead(ctx, caller, id)
if err != nil || !allowed {
    http.Error(w, "", http.StatusForbidden)       // an error denies
    return
}
```

**The invariant: the tenant predicate is enforced where it cannot be omitted** — a scoped
connection, row-level security, or a repository that requires a tenant. Every hand-written
query is a chance to forget it, and the failure is silent and total.

## How not to use it

**Authenticated but not authorised — the dominant flaw.**

```go
// WRONG — any logged-in user reads any decision.
func (h Handler) GetDecision(w http.ResponseWriter, r *http.Request) {
    if h.session.User(r) == nil {
        http.Error(w, "", http.StatusUnauthorized)
        return
    }
    d, _ := h.repo.Decision(r.Context(), DecisionID(r.PathValue("id")))
    writeJSON(w, d)
}
// Symptom: incrementing the id in the URL walks every tenant's decisions. The
// check proved identity and never mentioned the object.
```

**Trusting a client-asserted identity.**

```go
// WRONG — the tenant comes from the request.
tenant := r.Header.Get("X-Tenant-ID")
decisions, _ := h.repo.ForTenant(ctx, tenant)
// Symptom: changing one header reads another tenant. A user id, tenant id, role
// or price in a body, header, cookie or unsigned claim is INPUT, not fact.
```

**A per-query tenant filter.**

```go
// WRONG — correct in nine queries and forgotten in the tenth.
`SELECT * FROM decisions WHERE repository = $1 AND tenant_id = $2`   // ok
`SELECT * FROM decisions WHERE created_at > $1`                       // the leak
// Symptom: nothing distinguishes the two at review. Enforce structurally.
```

**Relying on the UI hiding the action, or on an unguessable URL.** Both are client-side, and
neither is a control.

**Verifying a token loosely.** Check signature, issuer, audience, expiry **and algorithm**.
Reject `none`, and never let the token choose its own verification key.

**Privilege widened temporarily and never narrowed.** A wildcard role, a debug bypass, a
`TODO: tighten this`. Grep for them.

**No test asserting denial.** A suite proving the owner can read **proves nothing about the
stranger**. Multi-tenant systems need fixtures with at least two tenants.

**Trade-off:** scoped queries and a central policy layer make some legitimate cross-tenant
operations — admin tooling, support access, reporting — genuinely awkward. Those paths need
their own deliberate, audited design rather than a bypass.

**Related:** `least-privilege.md`, `threat-modeling.md`, `security-review-and-testing.md`
