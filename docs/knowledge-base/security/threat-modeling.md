# Threat modeling

*Security · Derived from the global `sec-threat-modeling` skill — **binding on this project***

## What it is

Four questions asked before implementation: **what are you building** (data flow, plus the
trust boundaries), **what can go wrong** (STRIDE, per element), **what will you do about it**
(mitigate, eliminate, transfer or accept), and **did you do a good job** (every threat has a
control, an owner, or a recorded acceptance — and every control has a test).

## Why it exists

Because most serious vulnerabilities are **decisions, not typos**, and a decision is far
cheaper to change before it is implemented than after. A scanner finds the typo; nothing
except modelling finds the missing control.

It also exists to answer a specific trap: **security features are not secure features.**
Adding TLS, or encryption, or an auth library does not make software secure. Encrypting a
channel nobody attacks while the authorisation check is missing is effort spent for nothing,
and threat modelling is how you find out which is which.

## When to use it

Starting a new service or feature, changing how components trust each other, reviewing an
architecture, or whenever a trust boundary moves. And **proportionally** — a local CLI reading
local files does not need a formal model; a multi-tenant service holding a write-scoped forge
credential does.

The principles that decide most questions before any technique is reached for: assume external
systems are insecure; minimise the attack surface; least privilege; defence in depth; secure
defaults; **fail to a secure mode**; plan on failure; don't mix code and data; never depend on
obscurity alone; separation of duties; don't invent your own crypto or auth protocol.

## How to use it

Draw the boundaries first, because every arrow crossing one is where a threat lives.

```go
// RIGHT — the trust boundary is explicit in the code's shape. Nothing past this
// point treats the payload as anything but hostile.
func (h Handler) ForgeEvent(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
    if err != nil {
        http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
        return
    }
    if !verifySignature(r.Header.Get("X-Hub-Signature-256"), body, h.secret) {
        http.Error(w, "", http.StatusUnauthorized)   // verified BEFORE parsing
        return
    }
    event, err := parseEvent(body)                   // barricade: returns a domain type
    // ...
}
```

Then walk each element against STRIDE — **spoofing** (authentication), **tampering**
(integrity), **repudiation** (non-repudiation), **information disclosure** (confidentiality),
**denial of service** (availability), **elevation of privilege** (authorization).

**The invariant: apply STRIDE per element, not to the system.** "Can this endpoint be
spoofed?" yields findings; "is the system secure?" does not.

Boundaries people miss: between your service and a service you also wrote; between your code
and your own database when anything else can write to it; between a request handler and a
background job; **between tenants in a shared system**; between CI and production.

## How not to use it

**Verifying after parsing.**

```go
// WRONG — the parser ran on unauthenticated attacker-controlled bytes.
event, err := parseEvent(body)
if err != nil { http.Error(w, err.Error(), 400); return }
if !verifySignature(sig, body, secret) { http.Error(w, "", 401); return }
// Symptom: every parser bug is reachable by anyone. Verify, then parse.
```

**An error path that skips the check.**

```go
// WRONG — a policy lookup failure proceeds.
allowed, err := policy.MayRead(caller, id)
if err != nil {
    log.Warn("policy lookup failed", "err", err)
}
if allowed { return resource }        // allowed is false, but so is the deny
return resource                       // ...and it returns anyway
// Symptom: an exception routes around the control. Fail to a secure mode: deny.
```

**Accepting a risk silently.** Accepting one is legitimate; accepting it without writing down
the reason is not — nobody can revisit a decision nobody recorded.

**Modelling with no boundary crossings.** If nothing in the model crosses a boundary, the
model is wrong.

**Trusting a compatibility fallback.** An insecure protocol or default that shipped is one you
will support forever, and **where a fallback exists it *is* the security level**, because an
attacker will simply request it.

**Trade-off:** modelling costs time before any code exists, and the output is a list of things
that have not happened. That is also the only point at which the cheap fixes are still cheap.

**Related:** `authorization.md`, `input-validation.md`, `security-review-and-testing.md`
