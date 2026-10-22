# A02:2025 — Security Misconfiguration

*OWASP Top 10:2025 · External: OWASP*

## What it is

The software is capable of being secure and is not configured that way. Default credentials
left in place, debug endpoints exposed, verbose errors reaching callers, permissive CORS,
over-broad cloud roles, unnecessary features enabled, missing hardening on the container or
the platform. **Up from #5 in 2021 to #2** — the sharpest rise in the edition.

## Why it rose

Because infrastructure became code and configuration became the larger surface. A service
with no code defects can still be fully exposed by one wildcard IAM statement or one debug
flag, and those are edited by people under time pressure with no test asserting the safe
value.

It also rises because **the out-of-the-box configuration is the one most people run.** A
secure default protects everyone who never reads the documentation; an insecure default is a
vulnerability shipped to every adopter.

## When to check for it

- Any default that ships — **features off, TLS on, no default credentials, restrictive
  setting first.**
- Any debug or diagnostic path. Is it present in production builds? Is it authenticated?
- Container and process posture: non-root, read-only filesystem, dropped capabilities.
- Cloud roles and service accounts: specific actions and resources, never wildcards.
- Anything that was widened temporarily. Grep for `TODO: tighten`, wildcard roles, disabled
  verification.
- For this project specifically: the **policy evaluator's capability allowlist** is
  configuration, and getting it wrong lets tenant policy reach the network.

## How to use it

Make the safe value the default, and make it a **tested invariant** rather than a documented
intention.

```go
// RIGHT — the zero value is the safe one. An unset config enforces, not bypasses.
type Config struct {
    AllowPlaintext   bool          // false by default
    DebugEndpoints   bool          // false by default
    FailOpenOnError  bool          // false by default: errors deny
}
```

```go
// RIGHT — the restriction is a test, not a config option.
func TestCapabilitiesExcludeNetwork(t *testing.T) {
    caps := restrictedCapabilities()
    for _, b := range caps.Builtins {
        if b.Name == "http.send" {
            t.Fatal("http.send is reachable from tenant policy")
        }
    }
    if caps.AllowNet == nil || len(*caps.AllowNet) != 0 {
        t.Fatal("AllowNet must be empty, not nil")
    }
}
```

**The invariant: a security-relevant setting has a test asserting its safe value.** A default
nobody tests drifts at the first merge conflict.

## How not to use it

**An unsafe default.**

```go
// WRONG — the zero value bypasses the control.
type Config struct {
    EnforcePolicy bool      // false by default: an unset config enforces NOTHING
}
// Symptom: a missing config key silently disables the product's whole purpose.
// Invert the field so the default is safe.
```

**Verbose errors reaching the caller.**

```go
// WRONG
http.Error(w, err.Error(), 500)
// Output: pq: relation "decisions_v2" does not exist
// Symptom: the schema, the driver and the query are now public. See A10.
```

**A debug path that ships.**

```go
// WRONG — compiled into the production binary, unauthenticated.
mux.HandleFunc("/debug/dump-config", func(w http.ResponseWriter, _ *http.Request) {
    json.NewEncoder(w).Encode(cfg)         // includes the signing key
})
// Symptom: one unauthenticated request yields every secret.
```

**Permissive CORS, or a wildcard role.**

```json
// WRONG
{ "Effect": "Allow", "Action": "*", "Resource": "*" }
```

**Trade-off:** restrictive defaults generate support load — adopters hit a refusal before they
hit success, and every one is a question. That is the correct direction for the cost to fall.

**Related:** `../least-privilege.md`, `a10-mishandling-of-exceptional-conditions.md`,
`../../architecture/deployability.md`
