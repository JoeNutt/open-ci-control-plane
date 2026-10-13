# Strategy

*Behavioral · GoF · Authored for this project*

## What it is

A family of interchangeable algorithms is put behind one interface, and the context holds a
reference to whichever one it should use. Selecting behaviour becomes selecting an object.
The context knows *what* it needs done and nothing about *how*.

## When to use it

- The **how** varies and the **what** does not: comparison order, retry policy, pricing rule,
  compression choice, import extraction per language.
- The choice is made at runtime — per request, per tenant, per configuration.
- A conditional selecting an algorithm appears at one point and the set of algorithms grows.

## How to use it

In a language with first-class functions, **the strategy is a function**. Reach for an
interface only when the strategy needs multiple operations or its own state.

```go
// One operation: a function type is the whole pattern.
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

- **A class hierarchy for a one-method strategy in a language with closures or lambdas.**
  This is the most common over-application, and the one most cited in arguments that patterns
  have been absorbed by language features: pre-lambda Java and C++ needed the boilerplate,
  and modern code does not
  ([one statement of the argument](https://www.compiler.today/software-engineering/design-patterns-overrated-pragmatic-simplicity-2026)).
  Treat that as expert opinion, not settled fact — but the specific point about one-method
  strategies is sound.
- **Strategies with different interfaces in practice.** If callers must know which strategy
  is active to call it correctly, or pass different arguments, these are not interchangeable
  and the abstraction is false.
- **Letting strategies share mutable state through the context.** Then switching strategy
  changes behaviour in ways neither strategy describes.
- **One strategy, added for future flexibility.** An interface with a single implementation
  is indirection with no variation behind it. Add it when the second arrives.
- **Confusing it with State.** The structures are nearly identical. A strategy is chosen by
  the client and does not change itself; a state changes to another state as a consequence of
  handling an event.

**Trade-off:** the algorithm in use is not visible where it is called, so understanding
behaviour means finding where the strategy was selected.

**Related:** `state.md`, `template-method.md`, `../structural/bridge.md`
