# Decorator

*Structural · GoF · Authored for this project*

## What it is

An object is wrapped in another object that implements the same interface, adds behaviour
before or after delegating, and is itself wrappable. Because every layer presents the
wrapped type's interface, decorators compose to arbitrary depth and the client cannot tell
how many it holds.

## When to use it

- Behaviour must be added **per instance at runtime**, not per class at compile time —
  this request gets retries and tracing, that one does not.
- The combinations are many and inheritance would need a subclass per combination.
- Each added concern is genuinely orthogonal: logging, retry, caching, compression, rate
  limiting, authorisation.

## How to use it

Make the decorator implement the interface, hold the interface rather than a concrete type,
and keep each layer to **one** concern.

```go
type Evaluator interface {
    Evaluate(ctx context.Context, in Input) (Decision, error)
}

type timed struct {                      // one concern: timing
    next Evaluator
    obs  Observer
}

func (t timed) Evaluate(ctx context.Context, in Input) (Decision, error) {
    start := time.Now()
    d, err := t.next.Evaluate(ctx, in)
    t.obs.Record(time.Since(start), err)
    return d, err                        // the result is passed through untouched
}

// composition reads outermost-first
e := timed{next: retrying{next: core}}
```

The invariant: **a decorator preserves the contract it wraps.** Same semantics, same error
meanings, plus an effect. A layer that changes what the result means is not a decorator.

## How not to use it

- **Swallowing or reinterpreting errors in a layer.** A retry decorator that returns a
  zero value after exhausting attempts has converted a failure into a plausible answer, and
  the caller cannot tell. Pass the failure outward.
- **Order-dependent stacks with the order left implicit.** Retry outside caching retries a
  cache hit; caching outside retry caches a transient failure. The stack's order is a design
  decision and belongs where it is assembled, stated.
- **Deep stacks.** Six layers make a stack trace unreadable and a performance problem
  untraceable. If you need six, one of them is doing too little.
- **A decorator that needs a method the interface does not have**, prompting a downcast to
  the concrete type. That breaks substitutability and means the interface is wrong.
- **Using it where a parameter would do.** If every caller wants the behaviour, it is not a
  decoration; put it in the implementation.
- **Confusing it with Proxy.** Both wrap and both preserve the interface; a decorator adds
  behaviour to a call the client intends, while a proxy controls *whether and how* the call
  reaches the subject.

**Trade-off:** many small types, an assembly step that becomes its own thing to get right,
and indirection that obscures what is actually executing.

**Related:** `proxy.md`, `composite.md`, `adapter.md`
