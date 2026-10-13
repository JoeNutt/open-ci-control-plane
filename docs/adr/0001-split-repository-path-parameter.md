# ADR-0001 — Split the repository path parameter into owner and repo

**Status:** Accepted · **Date:** 2026-10-13 · **Diverges from:** specification §13, C-4, C-12, C-13

## Context

The signed specification gives three contracts a `{repository}` path parameter:

```
POST /v1/repositories/{repository}/commits/{headSha}/findings
PUT  /v1/repositories/{repository}/coverage/{suite}
GET  /v1/repositories/{repository}/telemetry
```

A repository full name is `owner/name` and contains a slash. A URI path parameter cannot
span path segments, so none of these three templates routes as written. Percent-encoding the
slash is not a workaround: many servers and proxies normalise or reject `%2F` in a path, and
the behaviour is not portable.

This was found while authoring `openapi.yaml`, where the defect is immediately visible
because the path template has to be machine-readable.

## Decision

Paths carry two parameters:

```
POST /v1/repositories/{owner}/{repo}/commits/{headSha}/findings
PUT  /v1/repositories/{owner}/{repo}/coverage/{suite}
GET  /v1/repositories/{owner}/{repo}/telemetry
```

**Request and response bodies are unchanged.** They keep the single `repository` field the
specification states, because a slash is legal in a body and there is no reason to diverge
there. The split exists only where the URI forces it.

## Consequences

The specification's §13 and `openapi.yaml` disagree on these three path templates, and
`openapi.yaml` is correct. Anyone reading §13 for the route shape must read this record with
it — which is why it names the contracts rather than describing the change generically. The
divergence is resolved, not outstanding; there is nothing to do.

The asymmetry between path and body is a wart: `{owner}/{repo}` in the URI, `repository` in
the payload. The alternative — splitting the body too — would have diverged from the baseline
for no technical reason, which is worse.
