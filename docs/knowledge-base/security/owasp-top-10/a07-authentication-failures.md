# A07:2025 — Authentication Failures

*OWASP Top 10:2025 · External: OWASP*

## What it is

Failures in establishing **who** the actor is, and in keeping that established. Credential
stuffing permitted, weak or unrotated secrets, a session that does not expire or rotate, a
token verified loosely, a recovery flow that bypasses the login, missing or bypassable
multi-factor. **Stable in rank**, renamed from 2021's *Identification and Authentication
Failures*.

## Why it persists

Because authentication has many edges and each one is a complete bypass. The login itself is
usually the best-defended path; the failures live in password reset, remember-me, token
refresh, service-to-service credentials, and the error messages that reveal which accounts
exist.

Keep the distinction from A01 clear: **authentication is who you are; authorization is what
you may do.** Conflating them is itself a common bug — checking the caller is logged in and
calling that an access control.

## When to check for it

- Is every token verified on **signature, issuer, audience, expiry and algorithm**? Is `none`
  rejected? Can the token choose its own key?
- Does the session rotate on privilege change and expire on an absolute as well as an idle
  timeout?
- Is there rate limiting and lockout on credential endpoints — and on the reset flow too?
- Do failures enumerate? "Invalid username" versus "invalid password", or a measurable timing
  difference, both reveal which accounts exist.
- For service-to-service: is the credential short-lived and audience-bound, or a long-lived
  shared secret?

## How to use it

For this project the relevant authentication is **machine-to-machine**: a forge webhook
signature and workload identity tokens.

```go
// RIGHT — verify before parsing, in constant time, on the raw body.
func (h Handler) ForgeEvent(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
    if err != nil { http.Error(w, "", 413); return }

    if !verifySignature(r.Header.Get("X-Hub-Signature-256"), body, h.secret) {
        http.Error(w, "", http.StatusUnauthorized)      // no detail
        return
    }
    event, err := parseEvent(body)                       // only now
    // ...
}
```

```go
// RIGHT — a token is checked on every claim that matters, including alg.
claims, err := jwt.Parse(raw,
    jwt.WithValidMethods([]string{"RS256"}),   // reject `none` and alg confusion
    jwt.WithIssuer(expectedIssuer),
    jwt.WithAudience(ourAudience),
    jwt.WithExpirationRequired(),
)
```

**The invariant: an unsigned or mis-signed request is indistinguishable from a forged one and
is treated as one.**

## How not to use it

**Verifying after parsing.**

```go
// WRONG — the parser ran on unauthenticated bytes.
event, _ := parseEvent(body)
if !verifySignature(sig, body, secret) { http.Error(w, "", 401); return }
// Symptom: every parser bug is reachable by anyone.
```

**Accepting any algorithm.**

```go
// WRONG — the token chooses how it is verified.
claims, err := jwt.Parse(raw, keyFunc)       // no WithValidMethods
// Symptom: alg=none, or HS256 signed with the public key, both verify.
```

**Enumerating accounts.**

```go
// WRONG
if inst == nil  { http.Error(w, "no such installation", 404); return }
if !sigOK       { http.Error(w, "bad signature", 401); return }
// Symptom: an attacker maps which installations exist. Return one generic
// refusal — and watch response timing, which enumerates just as well.
```

**A long-lived shared secret where a short-lived scoped token would do.** It must then be
stored, rotated and audited, and it is valid for as long as it takes anyone to notice.

**Trade-off:** generic refusals and strict token validation make legitimate integration
debugging harder, and short-lived credentials need a minting path that can itself fail. Both
are the price of not handing out a map.

**Related:** `../authorization.md`, `../secret-management.md`,
`a10-mishandling-of-exceptional-conditions.md`
