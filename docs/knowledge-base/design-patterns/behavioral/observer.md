# Observer

*Behavioral · GoF · Authored for this project*

## What it is

A subject maintains a list of dependents and notifies them when its state changes. Observers
register and deregister themselves, and the subject calls a known method on each without
knowing their concrete types. One change propagates to an open set of listeners.

## Why it exists

When one object's change must reach others, the direct approach has it call each of them, so
it depends on every listener and must be edited to add one. That couples a stable subject to a
changing set of consumers, often across boundaries that should not know about each other. The
pattern inverts the dependency: listeners register through one interface the subject owns,
and the subject notifies whoever is registered.

## When to use it

- A change must reach an **unknown or varying set** of interested parties.
- The subject must not depend on its listeners — that is the whole point, and it is what
  distinguishes this from calling them directly.
- Listeners come and go at runtime: UI bindings, cache invalidation, metrics, audit hooks.

## How to use it

Make deregistration as easy as registration, define whether notification is synchronous, and
state what happens when an observer panics.

```go
// RIGHT — registration returns its own removal; one observer cannot stop the rest.
type Unsubscribe func()

func (s *Subject) Observe(fn func(Event)) Unsubscribe {
	id := s.next
	s.next++
	s.obs[id] = fn
	return func() { delete(s.obs, id) }
}

func (s *Subject) emit(e Event) {
	for _, fn := range snapshot(s.obs) {
		notify(fn, e)
	}
}

func notify(fn func(Event), e Event) {
	defer recoverAndLog()
	fn(e)
}
```

`emit` iterates a snapshot so a handler may unsubscribe during notification, and `notify`
recovers so one panicking observer does not stop the rest.

The invariant: **the subject's correctness does not depend on any observer.** If a listener
failing changes the subject's outcome, the relationship is not observation and should be a
direct call.

## How not to use it

- **No way to deregister, or a registration whose removal is awkward.** This is the classic
  memory leak: the subject holds the observer forever, which holds whatever it closed over.
  Returning the unsubscribe handle at registration makes it hard to forget.

```go
// WRONG — registration with no way back.
func (s *Subject) Observe(fn func(Event)) {
	s.obs = append(s.obs, fn)
}
// Symptom: every closed session leaves its callback, and everything it closed over, on the subject.
```

- **Letting an observer's failure propagate into the subject.** One failing listener then
  prevents the rest from being notified, and which ones ran depends on iteration order.

```go
// WRONG — the first failing observer ends notification.
func (s *Subject) emit(e Event) error {
	for _, fn := range s.obs {
		if err := fn(e); err != nil {
			return err
		}
	}
	return nil
}
// Symptom: when the metrics observer fails, the audit observer may never hear; map order decides.
```

- **Mutating the observer list during notification** without iterating a snapshot — a
  listener that unsubscribes itself inside its own callback corrupts the iteration.

```go
// WRONG — handlers run against the live slice.
func (s *Subject) emit(e Event) {
	for i := range s.obs {
		s.obs[i](e)
	}
}
// Symptom: an observer that unsubscribes shifts the slice; the next one is skipped, or the last index panics.
```

- **Notification cascades.** An observer whose handler changes the subject triggers another
  notification. The trace is a loop through one emit, and the ordering is unspecified.

```go
// WRONG — an observer that changes the subject it observes.
s.Observe(func(e Event) {
	if e.Status == Failed {
		s.SetStatus(Retrying)
	}
})
// Symptom: one change emits another from inside its own notification, and observers see them in no fixed order.
```

- **Using it for a required step in a workflow.** If something *must* happen when the state
  changes, a listener is the wrong mechanism: nothing guarantees it is registered. Call it.

```go
// WRONG — recording the decision is left to whoever subscribed.
gate.Observe(func(e Event) { auditLog.Append(e.Decision) })
// Symptom: a wiring path that skips this line decides without an audit record, and nothing fails.
```

- **Synchronous notification on a hot path**, where an observer's latency becomes the
  subject's. Decide sync versus async deliberately and say which in the contract.

```go
// WRONG — every observer runs inline on the request path.
func (g *Gate) Decide(ctx context.Context, in Input) Decision {
	d := g.evaluate(ctx, in)
	g.emit(Event{Decision: d})
	return d
}
// Symptom: decision latency becomes the slowest observer's; a portal webhook's timeout becomes the gate's.
```

**Trade-off:** the set of things that happen on a change is not visible at the point of the
change, so cause and effect are separated in the code and in the stack trace.

**Related:** `mediator.md`, `../structural/decorator.md`
