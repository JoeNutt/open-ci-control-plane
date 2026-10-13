# Secret management

*Security · Derived from the global `sec-crypto-secrets` skill — **binding on this project***

## What it is

Where keys, tokens, passwords and connection strings live, how they rotate, and what must
never record them. **Secrets are what actually leaks** — the failure is almost never the
algorithm.

## Why it exists

Because a key stored beside the data it protects protects nothing, and because **git history
and container image layers are forever.** Deleting a commit does not unpublish a secret: any
secret ever committed must be treated as public and rotated.

It also exists because rotation designed later never happens. Supporting two valid keys
during a changeover has to be built in from the start, or the first rotation becomes an
outage nobody is willing to schedule.

## When to use it

Any time a credential is introduced, stored, passed, logged or formatted. Specifically:

- **Never commit secrets** — not in source, config, test fixtures, CI files, container
  images, infrastructure code, or comments.
- **Use a secrets manager** — the platform vault or KMS. Environment variables are
  acceptable at the edge; a `.env` in the repository is not.
- **Separate by purpose and environment.** One key, one job. Production keys never exist in
  development, and a developer laptop is not where production credentials live.
- **Rotate** on a schedule, after any suspected exposure, and when someone with access
  leaves.
- **Client-side secrets do not exist.** An API key in a mobile app, SPA or browser extension
  is published.

## How to use it

Keep secrets out of anything that formats, and redact where the value is **formatted**, not
where it is read.

```go
// RIGHT — the type cannot be printed by accident.
type Secret struct{ v string }

func (Secret) String() string          { return "[REDACTED]" }
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }
func (s Secret) Reveal() string        { return s.v }   // explicit, greppable

// log.Info("auth", "token", tok) now prints [REDACTED] whatever the call site does.
```

```go
// RIGHT — two valid keys during a changeover, so rotation is possible at all.
type SigningKeys struct {
    Current  Secret
    Previous Secret          // accepted for verification, never for signing
}

func (k SigningKeys) Verify(sig string, body []byte) bool {
    return verify(k.Current, sig, body) || verify(k.Previous, sig, body)
}
```

**The invariant: no secret reaches a log, a URL, an error message, a metric label or a crash
report.** Redact at the formatting point, because that is the only place that catches every
call site.

## How not to use it

**A secret in the repository.**

```go
// WRONG
const webhookSecret = "whsec_8f3a21c04d"      // committed
// Symptom: public the moment it is pushed, and present in history forever.
// Rotate it; do not rewrite the commit and consider it handled.
```

**A secret in a log or an error.**

```go
// WRONG — the token is now in the log aggregator and anyone who can read it.
log.Error("forge call failed", "token", token, "err", err)
return fmt.Errorf("auth failed for token %s: %w", token, err)
// Symptom: the secret has been copied to every system that ingests logs, and
// the error text may be returned to a caller.
```

**A secret in a URL.**

```go
// WRONG — query strings end up in access logs, proxies, browser history and
// Referer headers.
http.Get("https://api.example.com/bundles?key=" + apiKey)
// Symptom: the credential is recorded by every hop. Use a header.
```

**A key stored beside what it protects.**

```go
// WRONG — the encrypted column and its key in the same database.
`INSERT INTO secrets (ciphertext, key) VALUES ($1, $2)`
// Symptom: anyone who can read the data can decrypt it. The encryption bought
// nothing.
```

**One key for two purposes.** Signing bundles and encrypting decision records with the same
key means rotating one forces rotating the other, and a compromise of either is a compromise
of both.

**Rotation that was never designed.** A single-key verification path means a changeover is an
atomic cutover across every component simultaneously — which is why it never happens.

**Trade-off:** a redacting type means an explicit `Reveal()` at every genuine use, which is
noisier than a plain string. That noise is also the audit trail — every place the secret is
actually needed is greppable.

**Related:** `cryptography.md`, `least-privilege.md`, `error-handling-and-logging.md`
