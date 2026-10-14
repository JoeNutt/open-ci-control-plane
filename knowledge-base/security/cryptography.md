# Cryptography

*Security · Derived from the global `sec-crypto-secrets` skill — **binding on this project***

## What it is

Choosing and using cryptographic primitives. **Cryptography fails through misuse, not broken
maths** — the primitives are fine; the bugs are choosing the wrong one, using it in the wrong
mode, reusing a nonce, or protecting the key with the thing the key was supposed to protect.

## Why it exists

Because the failure modes are counter-intuitive and silent. AES-GCM with a reused nonce
destroys confidentiality *and* forgeability in one step, and nothing about the ciphertext
looks different. A byte-by-byte comparison of two MACs leaks the correct value through
timing while returning the right answer every time.

Two rules carry most of the weight: **encryption is not authentication** (ciphertext without
integrity can be tampered with), and **encryption is not authorization** (encrypted data
still needs an access check).

## When to use it

Any time data is hashed, encrypted, signed, or a random value is used for anything
security-bearing. The choices, named for the decision rather than to be memorised — check
current guidance when it matters:

| Need | Use | Never |
|---|---|---|
| Password storage | Argon2id; else scrypt or bcrypt | Any plain hash, salted or not |
| Data at rest / in transit | AEAD: AES-GCM, ChaCha20-Poly1305 | ECB; unauthenticated CBC; DES, 3DES, RC4 |
| Hashing / integrity | SHA-256, SHA-3, BLAKE2 | MD5, SHA-1 |
| Message authentication | HMAC-SHA-256, or the AEAD's own tag | Hand-built `hash(key + message)` |
| Signatures | Ed25519, ECDSA P-256, RSA-PSS ≥3072 | RSA PKCS#1 v1.5 for new work |
| Random for security | The platform **CSPRNG** | `math/rand`, time-seeded PRNGs |
| Key from a password | Argon2id / PBKDF2, high iterations | The password as a key |
| Key from a key | HKDF | Truncating or reusing a key for two purposes |
| Transport | TLS 1.3 (or 1.2), certificates verified | Disabled verification; plaintext fallback |

## How to use it

```go
// RIGHT — CSPRNG for anything security-bearing. crypto/rand.Read never returns an
// error: on a failed read it crashes the process. go doc crypto/rand.Read, Go 1.26.
func newDecisionID() string {
    var entropy [16]byte
    rand.Read(entropy[:])                     // crypto/rand
    return hex.EncodeToString(entropy[:])
}
```

```go
// RIGHT — a random secret string with no format of your own: rand.Text returns at
// least 128 bits in RFC 4648 base32. go doc crypto/rand.Text, Go 1.26.
token := rand.Text()
```

```go
// RIGHT — constant-time comparison for a MAC.
func verifySignature(header string, body, secret []byte) bool {
    mac := hmac.New(sha256.New, secret)
    mac.Write(body)
    want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(header), []byte(want))      // not ==
}
```

```go
// RIGHT — AEAD, with a fresh random nonce per message.
nonce := make([]byte, aead.NonceSize())
rand.Read(nonce)                                           // never fails; see above
sealed := aead.Seal(nonce, nonce, plaintext, aad)        // nonce prefixed, never reused
```

**The invariant: never invent an algorithm or protocol, and never implement a standard
primitive yourself.** Prefer a high-level library that picks the primitives for you.

## How not to use it

**A general-purpose PRNG for a security value.**

```go
// WRONG — predictable from a few outputs.
import "math/rand"
token := fmt.Sprintf("%d", rand.Int63())
// Symptom: an attacker who sees two tokens can compute the rest. Use crypto/rand.
```

**An error branch on crypto/rand.Read.**

```go
// WRONG — handles a failure the call cannot report.
if _, err := rand.Read(b); err != nil {
    return "", err
}
// Symptom: the branch never runs, so no test can reach it, and the signature now
// promises callers a failure that never comes — each of them grows its own dead
// branch. crypto/rand.Read never returns an error and crashes the process instead
// (go doc crypto/rand.Read, Go 1.26). Call it bare, and drop the error from the
// signature if it was there only for this.
```

**A byte-comparing equality check on a secret.**

```go
// WRONG — early exit leaks the value through timing.
if receivedMAC == expectedMAC { ... }
// Symptom: timing differences let an attacker recover the MAC byte by byte.
// Use hmac.Equal / subtle.ConstantTimeCompare.
```

**Encryption mistaken for integrity, or for authorization.**

```go
// WRONG — unauthenticated CBC. The ciphertext can be modified undetectably.
mode := cipher.NewCBCEncrypter(block, iv)
// Symptom: a bit-flipping attack changes the plaintext and nothing notices.
// Use an AEAD.
```

**A reused nonce.**

```go
// WRONG — a fixed nonce with a fixed key.
var nonce = make([]byte, 12)                 // all zeroes, every message
sealed := aead.Seal(nil, nonce, plaintext, nil)
// Symptom: AES-GCM nonce reuse destroys confidentiality AND allows forgery.
```

**Encoding mistaken for protection.** Base64 and hex are encodings. So is "XOR with a
constant".

**Hashing a password with a fast hash.** Passwords are **hashed, not encrypted** — with a
slow, memory-hard, per-password-salted function, parameters stored alongside and the cost
factor raised over time. A salted SHA-256 is not a password scheme.

**Disabling certificate verification, even behind a flag.** A disabled-verification path that
ships is the security level, because something will use it.

**Trade-off:** AEAD and a key-management story are more work than `encrypt(data, key)`, and
Argon2id costs real CPU per login. Both costs are the point.

**Related:** `secret-management.md`, `authorization.md`, `threat-modeling.md`
