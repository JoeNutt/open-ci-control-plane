# Observer

*Behavioral · GoF · Authored for this project*

## What it is

A subject maintains a list of dependents and notifies them when its state changes. Observers
register and deregister themselves, and the subject calls a known method on each without
knowing their concrete types. One change propagates to an open set of listeners.

## When to use it

- A change must reach an **unknown or varying set** of interested parties.
- The subject must not depend on its listeners — that is the whole point, and it is what
  distinguishes this from calling them directly.
- Listeners come and go at runtime: UI bindings, cache invalidation, metrics, audit hooks.

## How to use it

Make deregistration as easy as registration, define whether notification is synchronous, and
state what happens when an observer throws.

```go
type Unsubscribe func()

func (s *Subject) Observe(fn func(Event)) Unsubscribe {
    id := s.next; s.next++
    s.obs[id] = fn
    return func() { delete(s.obs, id) }       // the caller is handed its own removal
}

func (s *Subject) emit(e Event) {
    for _, fn := range snapshot(s.obs) {      // iterate a snapshot: handlers may unsubscribe
        func() {
            defer recoverAndLog()              // one bad observer must not stop the rest
            fn(e)
        }()
    }
}
```

The invariant: **the subject's correctness does not depend on any observer.** If a listener
failing changes the subject's outcome, the relationship is not observation and should be a
direct call.

## How not to use it

- **No way to deregister, or a registration whose removal is awkward.** This is the classic
  memory leak: the subject holds the observer forever, which holds whatever it closed over.
  Returning the unsubscribe handle at registration makes it hard to forget.
- **Letting an observer's failure propagate into the subject.** One throwing listener then
  prevents the rest from being notified, and which ones ran depends on iteration order.
- **Mutating the observer list during notification** without iterating a snapshot — a
  listener that unsubscribes itself inside its own callback corrupts the iteration.
- **Notification cascades.** An observer whose handler changes the subject triggers another
  notification. The trace is a loop through one emit, and the ordering is unspecified.
- **Using it for a required step in a workflow.** If something *must* happen when the state
  changes, a listener is the wrong mechanism: nothing guarantees it is registered. Call it.
- **Synchronous notification on a hot path**, where an observer's latency becomes the
  subject's. Decide sync versus async deliberately and say which in the contract.

**Trade-off:** the set of things that happen on a change is not visible at the point of the
change, so cause and effect are separated in the code and in the stack trace.

**Related:** `mediator.md`, `../structural/decorator.md`
