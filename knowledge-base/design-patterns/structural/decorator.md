# Decorator

*Structural · GoF · Authored for this project*

## What it is

An object is wrapped in another object that implements the same interface, adds behaviour
before or after delegating, and is itself wrappable. Because every layer presents the
wrapped type's interface, decorators compose to arbitrary depth and the client cannot tell
how many it holds.

## Why it exists

Adding a cross-cutting behaviour — timing, retry, caching — to some uses of a type and not
others, by editing the type or deriving variants of it, either multiplies types per
combination or loads every concern into one implementation that every caller pays for. The
decorator adds one behaviour by wrapping an object in another with the same interface, so
behaviours compose per instance at assembly and the wrapped type does not change.

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
// RIGHT — one concern per layer, and the result passed through untouched.
type Evaluator interface {
	Evaluate(ctx context.Context, in Input) (Decision, error)
}

type timed struct {
	next Evaluator
	obs  Observer
}

func (t timed) Evaluate(ctx context.Context, in Input) (Decision, error) {
	start := time.Now()
	d, err := t.next.Evaluate(ctx, in)
	t.obs.Record(time.Since(start), err)
	return d, err
}
```

```go
// RIGHT — assembly reads outermost-first, in one stated place.
evaluator := timed{next: retrying{next: core}, obs: obs}
```

The invariant: **a decorator preserves the contract it wraps.** Same semantics, same error
meanings, plus an effect. A layer that changes what the result means is not a decorator.

## How not to use it

- **Swallowing or reinterpreting errors in a layer.** A retry decorator that returns a
  zero value after exhausting attempts has converted a failure into a plausible answer, and
  the caller cannot tell. Pass the failure outward.

```go
// WRONG — the attempts run out, and the failure becomes an answer.
func (r retrying) Evaluate(ctx context.Context, in Input) (Decision, error) {
	for range r.attempts {
		if d, err := r.next.Evaluate(ctx, in); err == nil {
			return d, nil
		}
	}
	return Decision{}, nil
}
// Symptom: an outage reads as a zero Decision, and callers act on it.
```

- **Order-dependent stacks with the order left implicit.** Retry outside caching retries a
  cache hit; caching outside retry caches a transient failure. The stack's order is a design
  decision and belongs where it is assembled, stated.

```go
// WRONG — the cache is outermost because that line was written last.
evaluator := caching{next: retrying{next: core}}
// Symptom: a failure that outlasted the retries is cached, and served until the entry expires.
```

- **Deep stacks.** Six layers make a stack trace unreadable and a performance problem
  untraceable. If you need six, one of them is doing too little.

```go
// WRONG — six layers, several too thin to stand alone.
evaluator := tracing{next: timed{next: logging{next: retrying{next: caching{next: limiting{next: core}}}}}}
// Symptom: a panic's trace is six Evaluate frames deep before any policy code appears.
```

- **A decorator that needs a method the interface does not have**, prompting a downcast to
  the concrete type. That breaks substitutability and means the interface is wrong.

```go
// WRONG — the decorator reaches past the interface to the concrete type.
func (c caching) Evaluate(ctx context.Context, in Input) (Decision, error) {
	key := cacheKey(in, c.next.(*opaEvaluator).BundleRevision())
	return c.lookup(ctx, key, in)
}
// Symptom: putting timed between caching and the evaluator panics at the type assertion.
```

- **Using it where a parameter would do.** If every caller wants the behaviour, it is not a
  decoration; put it in the implementation.

```go
// WRONG — a wrapper that every assembly applies.
func NewEvaluator(core Evaluator) Evaluator {
	return validating{next: core}
}
// Symptom: no evaluator is ever built without it; the check belongs to the implementation.
```

- **Confusing it with Proxy.** Both wrap and both preserve the interface; a decorator adds
  behaviour to a call the client intends, while a proxy controls *whether and how* the call
  reaches the subject.

```go
// WRONG — named for measuring, but it decides whether the call happens.
func (m metered) Evaluate(ctx context.Context, in Input) (Decision, error) {
	if m.budget.IsExhausted(in.Tenant) {
		return Decision{}, ErrRateLimited
	}
	return m.next.Evaluate(ctx, in)
}
// Symptom: a reviewer looking for admission control never opens a type whose name says it measures.
```

**Trade-off:** many small types, an assembly step that becomes its own thing to get right,
and indirection that obscures what is actually executing.

**Related:** `proxy.md`, `composite.md`, `adapter.md`
