# Chain of Responsibility

*Behavioral · GoF · Authored for this project*

## What it is

Handlers are linked in a sequence. Each receives the request, decides whether it can handle
it, and either handles it or passes it to the next. The sender knows only the first handler,
so the set and order of handlers can change without the sender changing.

## Why it exists

When the right handler for a request depends on the request itself, and the set of candidate
handlers changes, the sender ends up holding a conditional that names every handler — so each
new handler edits the sender. The chain lets the sender address one entry point while each
handler decides whether the request is its own, so handlers are added, removed and reordered
where the chain is assembled rather than where requests are sent.

## When to use it

- The handler is **not known at the call site** and depends on the request's content or
  state.
- Handlers are genuinely independent and the set changes — middleware, validation stages,
  escalation tiers, event filters.
- More than one handler may legitimately contribute, or the first competent one should win and
  the rest should not run.

## How to use it

Decide explicitly whether the chain **stops at the first handler** or lets every handler see
the request, and make that visible in the return type rather than in a comment.

```go
// RIGHT — the stop rule is in the signature, and falling off the end is an error.
type Handler interface {
	Handle(ctx context.Context, r Request) (isHandled bool, err error)
}

func Dispatch(ctx context.Context, hs []Handler, r Request) error {
	for _, h := range hs {
		isHandled, err := h.Handle(ctx, r)
		if err != nil {
			return err
		}
		if isHandled {
			return nil
		}
	}
	return ErrUnhandled
}
```

An error ends the chain; a handler that takes responsibility ends it with `isHandled`.

The invariant: **falling off the end of the chain is an explicit outcome.** Silently
returning success when nothing handled the request is the pattern's signature bug.

## How not to use it

- **No terminal case.** An unhandled request that returns a zero value is a failure
  presented as an answer. Either a default handler ends the chain or the dispatcher reports
  that nothing handled it.

```go
// WRONG — nothing handled the request, and the caller is told nothing.
for _, h := range hs {
	if isHandled, err := h.Handle(ctx, r); err != nil || isHandled {
		return err
	}
}
return nil
// Symptom: an event kind no handler recognises is reported as processed.
```

- **Order-dependent chains with the order implicit.** If an authorisation handler must run
  before a caching handler, that is a rule, and it belongs where the chain is assembled,
  stated — not in whatever order the registrations happened to occur.

```go
// WRONG — each handler registers itself, so the order is init order.
func init() { Register(cachingHandler{}) }
// Symptom: renaming a file reorders init, and cached responses are served before authorisation runs.
```

- **Handlers that know about each other.** A handler checking "did the previous one run"
  has rebuilt the coupling the chain removed. Each decides on the request alone.

```go
// WRONG — the audit handler acts only if the auth handler ran.
func (h auditHandler) Handle(ctx context.Context, r Request) (bool, error) {
	if !r.HandledBy[authHandlerName] {
		return false, nil
	}
	return true, h.log.Record(ctx, r)
}
// Symptom: renaming the auth handler silently disables auditing.
```

- **Using it where a map would do.** If the handler is a direct function of one field, that
  is a lookup table, and a chain turns an O(1) dispatch into a linear scan with hidden
  ordering.

```go
// WRONG — a handler whose whole decision is one field compared with one value.
func (pushHandler) Handle(ctx context.Context, r Request) (bool, error) {
	if r.Event != "push" {
		return false, nil
	}
	return true, onPush(ctx, r)
}
// Symptom: every handler compares r.Event; a map keyed by event kind holds the same dispatch in one table.
```

- **Mutating the request as it travels, without saying so.** A later handler then depends on
  an earlier one's edits, and the chain becomes a pipeline whose stages cannot be reordered
  or omitted.

```go
// WRONG — a handler edits the request for the ones after it.
func (normalisingHandler) Handle(ctx context.Context, r *Request) (bool, error) {
	r.Path = strings.ToLower(r.Path)
	return false, nil
}
// Symptom: the routing handler matches only when this one ran first, and neither says so.
```

**Trade-off:** no single place states which handler will run, so tracing a request means
reading the whole chain and its assembly order.

**Related:** `command.md`, `../structural/decorator.md` — its structural cousin
