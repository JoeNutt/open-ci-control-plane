# Resource-oriented HTTP

*Technology · Authored for this project — advisory. Verified 2026-10-18 against the sources listed at the end*

## What it is

Each thing the service holds has a URI; the HTTP method says what happens to it (`GET` safe,
`PUT` and `DELETE` idempotent, `POST` neither); the status code says how it went; the body is a
JSON representation. That is Richardson level 2. "REST" proper, in Fielding's sense, also
requires hypermedia controls, which this project declines (SAD D-21) — so the surface is called
resource-oriented, not REST. Semantics come from RFC 9110, caching from RFC 9111, error bodies
from RFC 9457, and a safe read carrying a body from the `QUERY` method, RFC 10008 (June 2026).

## Why it exists

It reuses machinery every client, proxy, gateway, log and rate limiter already understands.
Retries are safe where the method says so; caches key on the URL; a 404 means the same thing to
curl, a CI step and an API gateway. Without it, every endpoint invents its own outcome encoding
and every intermediary is blind to it.

## When to use it

- The callers are machines with **one job each**: a forge delivering an event, a CI step
  uploading a file, an operator replacing a binding, a portal reading one aggregate.
- The surface is mostly **commands**, each with its own limits, credential and idempotency.
- The status code is part of the contract — deliberate 404-not-403, 410 for retention, 204 for
  "nothing evaluated".
- A third party fixes the request shape.

## How to use it

Bound the body before decoding, give each resource its own limit, and map outcomes to status
codes from one table.

```go
// RIGHT — the limit is enforced before the decoder reads a byte.
func (h bindingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBindingBytes)
	request, err := parseBindingRequest(r.Body)
	if err != nil {
		writeProblem(w, problemFor(err))
		return
	}
	outcome, err := h.replace.Execute(r.Context(), r.PathValue("bindingId"), request)
	writeOutcome(w, outcome, err)
}
```

**Invariant:** one outcome, one status, everywhere. The mapping lives in a single table in the
driving adapter; the application layer returns domain errors and never imports `net/http`.

## How not to use it

**Encoding failure in a 200.**

```go
// WRONG — the status says success; the outcome is in a field no intermediary reads.
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "unknown policy"})
```

Symptom: retries, gateways and dashboards count refusals as successes. Instead use the status
and an RFC 9457 problem body.

**Checking size after decoding.**

```go
// WRONG — the whole body is in memory before the limit is looked at.
body, _ := io.ReadAll(r.Body)
if len(body) > maxBindingBytes { http.Error(w, "too large", http.StatusRequestEntityTooLarge) }
```

Symptom: a large body costs memory before it is refused. Use `MaxBytesReader` first.

**Making the client walk the graph.** A screen needing a list and each item's children, served
as one call per item, is the round-trip problem GraphQL was built for.

```go
// WRONG — N+1 round trips, driven from the client.
for _, id := range listDecisionIDs(repo) { decisions = append(decisions, getDecision(id)) }
```

Instead model the view the client actually needs as a resource of its own — an aggregate such as
`/repositories/{repository}/telemetry` is a resource, not a REST violation — or embed the
children in the collection's representation. Reach for GraphQL only when those views multiply
across independent clients.

**Trade-off:** the server, not the client, decides response shape, so a new view is a new
resource and a release.

**Related:** `graphql.md`, `../security/README.md`, `../architecture/ports-and-adapters.md`.

Sources: RFC 9110, RFC 9111, RFC 9457 (obsoletes 7807), RFC 10008 (HTTP QUERY, Proposed Standard,
June 2026); draft-ietf-httpapi-idempotency-key-header-07 (still a draft); Fielding, "REST APIs must
be hypertext-driven" (2008); OpenAPI 3.2.0 (September 2025; adds QUERY and streaming media types).
