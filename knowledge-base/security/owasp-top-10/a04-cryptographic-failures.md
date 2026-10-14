# A04:2025 — Cryptographic Failures

*OWASP Top 10:2025 · External: OWASP*

## What it is

Sensitive data exposed because cryptography was absent, wrong, or misused. Plaintext
transmission, a weak or outdated algorithm, an unauthenticated mode, a reused nonce, a
predictable random value, a key stored beside what it protects, a password hashed with a fast
hash. **Down from #2 in 2021 to #4** — a fall in rank, not in severity.

## Why it ranks where it does

The fall reflects the rise of misconfiguration and supply chain rather than any improvement
in crypto practice. The category persists because **the failures are silent**: AES-GCM with a
reused nonce produces ciphertext that looks identical to correct output, and a timing-unsafe
MAC comparison returns the right answer every single time while leaking the value.

The framing worth carrying: cryptography fails through **misuse, not broken maths.**

## When to check for it

- Any data classified as credentials, personal, financial or health — **classify before
  protecting**, because the handling follows from the classification.
- Anything in transit, including internal service-to-service traffic.
- Anything at rest, **including backups, exports, caches and search indexes** — the copies are
  where breaches happen.
- Any random value that security depends on: tokens, session ids, nonces, IVs, salts, reset
  links, keys.
- Any comparison of a secret, MAC or hash.

## How to use it

The standard is `../cryptography.md`. Here, the three checks that catch most of it.

```go
// RIGHT — CSPRNG for anything security-bearing.
var entropy [16]byte
rand.Read(entropy[:])        // crypto/rand; never returns an error — see ../cryptography.md
```

```go
// RIGHT — constant-time comparison.
return hmac.Equal([]byte(received), []byte(expected))
```

```go
// RIGHT — AEAD with a fresh nonce per message.
nonce := make([]byte, aead.NonceSize())
rand.Read(nonce)
sealed := aead.Seal(nonce, nonce, plaintext, aad)
```

**The invariant: encryption is not authentication, and not authorization.** Use an AEAD, and
still check access.

## How not to use it

**A general-purpose PRNG.**

```go
// WRONG
import "math/rand"
token := strconv.FormatInt(rand.Int63(), 36)
// Symptom: predictable from a few observed outputs.
```

**A byte-comparing equality on a secret.**

```go
// WRONG — early exit leaks the value through timing.
if received == expected { ... }
// Symptom: the MAC is recoverable byte by byte.
```

**A fast hash for a password.**

```go
// WRONG — "salted SHA-256" is not a password scheme.
sum := sha256.Sum256(append(salt, password...))
// Symptom: a stolen database is cheap to attack. Passwords are hashed with a
// slow, memory-hard function — Argon2id — with parameters stored alongside.
```

**Disabled verification, even behind a flag.**

```go
// WRONG
tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
// Symptom: a path that ships is the security level, because something will use it.
```

**Trade-off:** AEAD plus a real key-management story is substantially more work than
`encrypt(data, key)`, and Argon2id costs measurable CPU per login. Both costs are the point.

**Related:** `../cryptography.md`, `../secret-management.md`, `a02-security-misconfiguration.md`
