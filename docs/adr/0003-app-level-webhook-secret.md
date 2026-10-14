# ADR-0003 — One webhook secret per App, not per installation

**Status:** Accepted · **Date:** 2026-10-14 · **Diverges from:** specification §13, contract C-1

## Context

C-1 states the authentication for the forge event sink:

> `X-Hub-Signature-256` HMAC over the raw body, verified with **the installation's webhook
> secret** *before the body is parsed*.

Those two halves cannot both hold. A GitHub webhook delivery carries no installation identifier
in any header — the installation appears only inside the JSON payload, as `installation.id`. So
selecting a per-installation secret requires parsing the body first, which is exactly what the
same sentence forbids, and what FR-1 requires:

> FR-1 — The control plane verifies the webhook HMAC signature before parsing the body.

Verified against GitHub's documentation rather than assumed: a GitHub App has **one** webhook
configuration — one URL and one secret — set on the app's registration page, and the
documentation describes no per-installation webhook secret. The signature arrives as
`X-Hub-Signature-256`, prefixed `sha256=`, an HMAC-SHA256 hex digest over the raw body, and
GitHub explicitly asks for a constant-time comparison rather than `==`.

## Decision

The secret is **the App's**, held once by the deployment, and verification uses it for every
delivery. C-1's "the installation's" is read as "the App's".

Verification therefore genuinely precedes parsing, with no lookup between the two, which is the
property FR-1 exists for.

## Consequences

FR-1 and FR-2 are implementable as written. The verification stage needs the raw body, the
signature header and one secret, and nothing else — no store, no network call, no parse.

**The isolation C-1 implied is weaker than its wording suggested, and the wording was the only
place it was claimed.** One secret for the App means any installation's delivery verifies
against the same key, so a signature proves the request came from GitHub's webhook pipeline for
this App — not that it came from a particular installation. Authorising the *installation*
therefore cannot rest on the signature and has to happen after parsing, against
`installation.id`, which is where C-1's `404 No installation matches` already sits. That is a
separate concern on a later card, and it is the reason this record exists rather than a silent
re-reading.

Rotating the secret rotates it for every installation at once. GitHub's webhook settings accept
one secret at a time, so a rotation is a brief window in which deliveries signed with the
previous secret fail closed and GitHub retries. Accepting two secrets during a rotation would
be a deliberate addition, not something C-1 asks for, and is not built here.
