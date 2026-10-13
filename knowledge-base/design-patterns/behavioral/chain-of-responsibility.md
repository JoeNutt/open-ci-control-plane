# Chain of Responsibility

*Behavioral · GoF · Authored for this project*

## What it is

Handlers are linked in a sequence. Each receives the request, decides whether it can handle
it, and either handles it or passes it to the next. The sender knows only the first handler,
so the set and order of handlers can change without the sender changing.

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
type Handler interface {
    // Handle returns handled=true when it has taken responsibility and the
    // chain must stop. The contract is in the signature, not in prose.
    Handle(ctx context.Context, r Request) (handled bool, err error)
}

func Dispatch(ctx context.Context, hs []Handler, r Request) error {
    for _, h := range hs {
        handled, err := h.Handle(ctx, r)
        if err != nil { return err }          // an error ends the chain
        if handled { return nil }
    }
    return ErrUnhandled                       // falling off the end is not success
}
```

The invariant: **falling off the end of the chain is an explicit outcome.** Silently
returning success when nothing handled the request is the pattern's signature bug.

## How not to use it

- **No terminal case.** An unhandled request that returns a zero value is a failure
  presented as an answer. Either a default handler ends the chain or the dispatcher reports
  that nothing handled it.
- **Order-dependent chains with the order implicit.** If an authorisation handler must run
  before a caching handler, that is a rule, and it belongs where the chain is assembled,
  stated — not in whatever order the registrations happened to occur.
- **Handlers that know about each other.** A handler checking "did the previous one run"
  has rebuilt the coupling the chain removed. Each decides on the request alone.
- **Using it where a map would do.** If the handler is a direct function of one field, that
  is a lookup table, and a chain turns an O(1) dispatch into a linear scan with hidden
  ordering.
- **Mutating the request as it travels, without saying so.** A later handler then depends on
  an earlier one's edits, and the chain becomes a pipeline whose stages cannot be reordered
  or omitted.

**Trade-off:** no single place states which handler will run, so tracing a request means
reading the whole chain and its assembly order.

**Related:** `command.md`, `decorator.md` is its structural cousin, `../structural/decorator.md`
