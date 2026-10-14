# Strategy

*Behavioral · GoF · Authored for this project*

## What it is

A family of interchangeable algorithms is put behind one interface, and the context holds a
reference to whichever one it should use. Selecting behaviour becomes selecting an object.
The context knows *what* it needs done and nothing about *how*.

## Why it exists

When one step of an operation has several interchangeable algorithms — retry timing,
comparison order, extraction per language — writing them inline leaves the caller with a
conditional that grows with every variant and mixes many hows into code that only cares about
the what. The pattern gives each algorithm its own value behind one interface, so the caller
is written once and a new algorithm is added without editing it.

## When to use it

- The **how** varies and the **what** does not: comparison order, retry policy, pricing rule,
  compression choice, import extraction per language.
- The choice is made at runtime — per request, per tenant, per configuration.
- A conditional selecting an algorithm appears at one point and the set of algorithms grows.

## How to use it

In a language with first-class functions, **the strategy is a function**. Reach for an
interface only when the strategy needs multiple operations or its own state.

```go
// RIGHT — one operation: a function type is the whole pattern.
type Backoff func(attempt int) time.Duration

func Retry(ctx context.Context, b Backoff, op func() error) error { /* ... */ }

// Several operations, or state: an interface earns its place.
type ImportExtractor interface {
	Claims(path string) bool
	Extract(path string, src []byte) ([]Ref, error)
}
```

The invariant: **the context does not inspect which strategy it holds.** A type switch over
strategies rebuilds the conditional the pattern removed.

## How not to use it

- **A type hierarchy for a one-method strategy in a language with closures.** This is the
  most common over-application, and the one most cited in arguments that patterns have been
  absorbed by language features: pre-lambda Java and C++ needed the boilerplate, and modern
  code does not
  ([one statement of the argument](https://www.compiler.today/software-engineering/design-patterns-overrated-pragmatic-simplicity-2026)).
  Treat that as expert opinion, not settled fact — but the specific point about one-method
  strategies is sound.

```go
// WRONG — an interface and a type for what is one function.
type BackoffStrategy interface {
	Delay(attempt int) time.Duration
}

type exponentialBackoff struct{}

func (exponentialBackoff) Delay(attempt int) time.Duration {
	return time.Duration(1<<attempt) * time.Second
}
// Symptom: each new policy is a type and a method where one func(int) time.Duration would do.
```

- **Strategies with different interfaces in practice.** If callers must know which strategy
  is active to call it correctly, or pass different arguments, these are not interchangeable
  and the abstraction is false.

```go
// WRONG — the caller must know which extractor it holds to call it correctly.
if _, isGoExtractor := extractor.(goExtractor); isGoExtractor {
	refs, err = extractor.Extract(path, src)
} else {
	refs, err = extractor.Extract(filepath.Base(path), src)
}
// Symptom: adding an extractor edits this call site — the edit the strategy existed to prevent.
```

- **Letting strategies share mutable state through the context.** Then switching strategy
  changes behaviour in ways neither strategy describes.

```go
// WRONG — the strategy keeps its running state on the context.
func (r *Retrier) linear(attempt int) time.Duration {
	r.lastDelay += time.Second
	return r.lastDelay
}
// Symptom: switching to exponential mid-run starts from whatever linear left in r.lastDelay.
```

- **One strategy, added for future flexibility.** An interface with a single implementation
  is indirection with no variation behind it. Add it when the second arrives.

```go
// WRONG — an interface with one implementation and no second in sight.
type PriceRule interface {
	Price(o Order) Money
}

type standardPriceRule struct{}
// Symptom: every reader follows the interface to find there is only ever one answer.
```

- **Confusing it with State.** The structures are nearly identical. A strategy is chosen by
  the client and does not change itself; a state changes to another state as a consequence of
  handling an event.

```go
// WRONG — a "strategy" that chooses its own successor.
type Backoff interface {
	Delay(attempt int) (time.Duration, Backoff)
}
// Symptom: after the first retry the policy running is not the one the caller configured; this is State.
```

**Trade-off:** the algorithm in use is not visible where it is called, so understanding
behaviour means finding where the strategy was selected.

**Related:** `state.md`, `template-method.md`, `../structural/bridge.md`
