# Least privilege

*Security · Derived from the global `sec-authz-least-privilege` skill — **binding on this project***

## What it is

Every actor — human, process, service, container, token — gets the **minimum rights needed,
for the minimum time**. Distinct from authorization, which decides whether a request is
allowed; this decides what the thing doing the work is capable of if it is compromised.

## Why it exists

Because controls get breached and bugs ship. The question least privilege answers is
**what is the blast radius if this component is fully compromised?** — and it is the only
control that still works after every other one has failed.

It also exists because the damaging configurations are usually temporary fixes nobody
removed. A role granting `*` on `*` is the single most damaging misconfiguration available,
and it almost never arrives deliberately.

## When to use it

Whenever a process, credential, token or role is created or widened. Specifically:

- **Processes** — not root. Drop privileges immediately if elevation is only needed at
  startup, and never regain them. Containers: non-root user, read-only filesystem, dropped
  capabilities, no privileged mode.
- **Database accounts** — the application account does not need schema rights. Separate
  accounts for migrations and runtime; read-only where only reads happen.
- **Service accounts and cloud roles** — specific resources and actions, never wildcards.
- **Tokens and API keys** — narrow scopes, short lifetimes, bound to an audience. Prefer
  short-lived credentials that rotate automatically.
- **Network and filesystem** — minimum permissions, minimum surface, bind to localhost unless
  remote access is required.

And **separation of duties**: the component requesting a sensitive action should not be the one
approving it.

## How to use it

```go
// RIGHT — the forge credential is scoped per installation and per repository,
// minted on demand with a short life. A compromised handler cannot reach beyond
// the repository it is serving.
func (f Forge) tokenFor(ctx context.Context, inst InstallationID, repo Repository) (Token, error) {
    return f.app.InstallationToken(ctx, inst, github.TokenOptions{
        Repositories: []string{repo.Name},
        Permissions: map[string]string{
            "checks":        "write",     // exactly what the enforcer needs
            "pull_requests": "read",
        },
    })
}
```

```go
// RIGHT — separation of duties. The enforcement path cannot grant itself an
// exemption; break-glass runs on a different credential on a different listener.
type Server struct {
    enforce   *http.Server    // the forge credential
    breakGlass *http.Server   // a separate credential, named release owners only
}
```

```go
// RIGHT — the policy evaluator's capabilities are an allowlist, so tenant policy
// cannot reach the network from inside the process.
rego.Capabilities(restrictedCapabilities())    // http.send removed, AllowNet empty
```

**The invariant: privilege is scoped to a resource and an action, and it expires.** A
credential with no expiry and no scope is a permanent liability.

## How not to use it

**A wildcard role.**

```json
// WRONG — the most damaging misconfiguration available.
{ "Effect": "Allow", "Action": "*", "Resource": "*" }
```

```go
// Symptom: a single compromised component owns the account. This almost always
// began as "temporarily, to unblock the deploy".
```

**One credential for everything.**

```go
// WRONG — the enforcement path holds a credential that can also grant exemptions.
srv := &http.Server{Handler: mux}      // one listener
mux.Handle("/v1/forge/github/events", eventHandler)
mux.Handle("/v1/breakglass-grants", grantHandler)   // same credential
// Symptom: compromising the webhook path yields the ability to exempt any
// repository from enforcement. No separation of duties.
```

**Running as root because it was easier.**

```dockerfile
# WRONG
FROM golang:1.26
COPY controlplane /usr/local/bin/
ENTRYPOINT ["controlplane"]
# Symptom: uid 0, writable filesystem, full capability set. A file-write bug
# becomes container escape surface.
```

```dockerfile
# RIGHT
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --chown=nonroot:nonroot controlplane /controlplane
ENTRYPOINT ["/controlplane"]
```

**One database account for migrations and runtime.** The runtime account then holds DDL
rights it never uses, so a SQL injection becomes schema destruction rather than data
disclosure.

**A long-lived token where a short-lived one would do.** Long-lived secrets must be stored,
rotated and audited; short-lived ones expire before an attacker has finished using them.

**Trade-off:** narrow scopes mean more credentials to provision and more failures that look
like permission bugs during development. The compensation only becomes visible during an
incident.

**Related:** `authorization.md`, `secret-management.md`, `threat-modeling.md`
