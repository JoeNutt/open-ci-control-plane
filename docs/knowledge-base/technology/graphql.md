# GraphQL

*Technology · Authored for this project — advisory. Verified 2026-10-18 against the sources listed at the end*

## What it is

A query language and execution engine over a typed schema. The client sends one document
naming exactly the fields it wants across related types; the server resolves each field with a
resolver function and returns a JSON object of the same shape. The specification (September
2025 edition, the first since October 2021) defines the language, type system and execution —
**not** the transport. HTTP transport is a separate document, *GraphQL over HTTP*, still a
working draft. Usually one endpoint, `POST /graphql`; `GET` is optional and may never run a
mutation.

## Why it exists

A client rendering one screen from many related resources pays for it in round trips — fetch a
list, then fetch each item's children — and in fields it did not ask for. The alternative is a
bespoke endpoint per screen, which multiplies endpoints and couples the server's release to every
client's layout. GraphQL moves the shape of the response to the client: one request, one
traversal of the graph, only the fields named.

## When to use it

- Many **independent clients** with different views over the **same graph of related
  entities**, where the server team cannot keep up with per-client endpoints.
- Reads dominate, and a typical read crosses two or more relationships.
- The clients are developers you can ship a schema to, and a typed client generator pays off.

It is the wrong fit when the surface is mostly commands, when the callers are fixed and few,
when a third party dictates the request shape (webhooks), or when the HTTP status code carries
meaning a caller relies on.

## How to use it

Bound demand before execution, authorise in every resolver, and carry failure in `errors` with
a machine-readable code — because the status code will not carry it.

```go
// RIGHT — cost bounded before any resolver runs; one transport; errors carry a code.
srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers}))
srv.AddTransport(transport.POST{})
srv.Use(extension.FixedComplexityLimit(maxOperationCost))

func (r *queryResolver) Decision(ctx context.Context, id string) (*model.Decision, error) {
	decision, err := r.decisions.ReadableBy(ctx, caller(ctx), id)
	if errors.Is(err, domain.ErrNotReadable) {
		return nil, &gqlerror.Error{Message: "not found",
			Extensions: map[string]any{"code": "NOT_FOUND"}}
	}
	return decision, err
}
```

**Invariant:** every resolver that returns a tenant's object checks the caller can read *that
object*. Reaching it through a parent the caller could read proves nothing.

## How not to use it

**Treating HTTP status as the outcome.** Under the over-HTTP draft, execution errors with partial
data return a 2xx (the draft says `294`, and elsewhere `200`). A 404, 410, 204 or 429 in a
resource contract has no equivalent.

```go
// WRONG — the client branches on the status, which is 2xx for a refused read.
if resp.StatusCode == http.StatusNotFound { return ErrUnknownDecision }
```

Symptom: a missing or forbidden object reads as success with a `null`. Instead branch on
`errors[].extensions.code` — and define that code set yourself, because nothing standard does.

**No demand control.** Depth, aliases and nested lists multiply cost inside one HTTP request.

```go
// WRONG — rate limited per request, so one request asks for 500 aliased decisions.
srv := handler.New(schema)
mux.Handle("/graphql", perRequestRateLimit(srv))
```

Symptom: one request pins a CPU. Instead bound complexity per operation and rate-limit by cost
(OWASP GraphQL Cheat Sheet: "HTTP request counts alone do not account for expensive queries").

**Authorising only at the endpoint.**

```go
// WRONG — the token was checked once; the edge to another tenant's object was not.
func (r *repositoryResolver) Decisions(ctx context.Context, repo *model.Repository) ([]*model.Decision, error) {
	return r.store.DecisionsFor(repo.ID)
}
```

Symptom: an IDOR through a relationship. Check on edges and nodes, and drop a generic `node(id)`
lookup unless every type behind it checks.

**Tunnelling file upload.** Upload is not in the GraphQL spec or the over-HTTP draft; the common
multipart spec is a community document whose `multipart/form-data` requests skip CORS
preflight, so it needs its own CSRF control. Upload over a plain HTTP endpoint instead.

**Trade-off:** you give up URL-keyed HTTP caching, per-endpoint body limits and status-code
semantics, and take on cost analysis, persisted operations and resolver-level authorisation.

**Related:** `resource-oriented-http.md`, `../architecture/ports-and-adapters.md`,
`../security/README.md`.

Sources: spec.graphql.org/September2025 and graphql.org/blog/2025-09-08-september-edition;
http-spec.graphql.org/draft (status codes, GET, CSRF); graphql.org/learn/performance and
/learn/caching; cheatsheetseries.owasp.org/cheatsheets/GraphQL_Cheat_Sheet.html;
gqlgen.com/reference/complexity (gqlgen v0.17.93, June 2026, pre-1.0);
github.com/jaydenseric/graphql-multipart-request-spec.
