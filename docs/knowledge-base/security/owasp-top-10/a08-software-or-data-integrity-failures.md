# A08:2025 — Software or Data Integrity Failures

*OWASP Top 10:2025 · External: OWASP*

## What it is

Code or data trusted **without verifying its provenance or integrity**. An unsigned update
applied, a plugin loaded from an unverified source, serialised state accepted from a client,
a CI artifact promoted without a check that it is the one that was built, a cache or message
trusted because of where it arrived from. **Stable at #8.**

## Why it is distinct from A03

A03 is about the chain that produces your software — who you depend on and what your pipeline
does. A08 is about the **verification step at the point of use**: whether the consuming system
checks that what it received is what was intended. The same incident can involve both; the
controls are different. A03 says sign it; A08 says verify it before you run it.

## When to check for it

- Does anything load code, policy or configuration from outside the binary? Is its signature
  verified, and does verification **failure** stop the load?
- Is any client-supplied state deserialised and trusted — a resumable token, a cursor, a
  cached object?
- Is an artifact promoted between environments by identity (digest) or by name (tag)?
- For this project: **policy bundles.** Signature plus file-set verification is exactly this
  category, and the operational detail is that a failed verification leaves the previous
  bundle enforcing **silently** — so the alert is the control.

## How to use it

Verify, fail closed to the last known good, and alert.

```go
// RIGHT — verification failure keeps the previous bundle and raises an alert.
func (l *Loader) activate(ctx context.Context, b bundle.Bundle) error {
    if err := l.verifier.Verify(b); err != nil {
        l.alerts.Raise(ctx, AlertActivationFailed, err)   // the alert IS the control
        return fmt.Errorf("verify bundle %s: %w", b.Revision, err)
    }
    l.current.Store(&b)
    return nil
}
```

```go
// RIGHT — promote by digest, not by tag. The identity is the content.
image := "ghcr.io/joenutt/controlplane@sha256:9f1c..."
```

```go
// RIGHT — client state is re-derived, never trusted. The cursor is opaque and
// signed, so a tampered one is rejected rather than obeyed.
cursor, err := cursors.Open(req.Cursor, c.TenantID)   // tenant-bound, MAC'd
if err != nil { return ErrInvalidCursor }
```

**The invariant: verification failure is loud and changes nothing.** A verifier whose failure
path proceeds is decoration.

## How not to use it

**Verification that logs and continues.**

```go
// WRONG
if err := verifier.Verify(b); err != nil {
    log.Warn("bundle signature invalid, loading anyway", "err", err)
}
l.current.Store(&b)
// Symptom: the signature is decorative. An attacker who can reach the bundle
// server can replace policy.
```

**Silent fallback with no alert.**

```go
// WRONG — correct behaviour, no signal.
if err := verifier.Verify(b); err != nil {
    return err                      // previous bundle keeps enforcing, quietly
}
// Symptom: stale policy enforces indefinitely and nobody knows. The fail-closed
// behaviour is right; without the alert it is indistinguishable from working.
```

**Trusting deserialised client state.**

```go
// WRONG — the client hands back state and it is believed.
var cur Cursor
json.Unmarshal(decoded, &cur)
rows := repo.After(ctx, cur.LastID, cur.TenantID)    // tenant from the client
// Symptom: editing the cursor reads another tenant. This is also A01.
```

**Promoting by tag.** A mutable tag means the artifact tested is not provably the artifact
deployed.

**Trade-off:** signing and verifying adds a key to manage, a rotation story, and a failure
mode where a legitimate bundle is refused. For this project the signing itself is a
cycle-1 *Should* — but **file-set verification and the activation alert stay Must**, because
they are what make the fail-closed behaviour real.

**Related:** `a03-software-supply-chain-failures.md`, `../cryptography.md`,
`a09-logging-and-alerting-failures.md`
