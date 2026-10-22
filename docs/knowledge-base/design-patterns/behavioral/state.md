# State

*Behavioral · GoF · Authored for this project*

## What it is

An object's behaviour is delegated to a state object, and changing state means swapping that
object. Each state class implements the same interface and handles events in its own way,
including deciding which state comes next. The conditional over a state field disappears into
polymorphism.

## Why it exists

When behaviour depends on which state an object is in, the usual form is a conditional over a
state field in every method, so adding a state means finding and editing each of them — and a
missed one leaves an event unhandled in that state. The pattern gives each state its own type
implementing every event, so a state's behaviour lives together and a state missing an event
does not compile.

## When to use it

- A **conditional over the same state field appears in several methods**, and adding a state
  means finding and editing all of them. This is the symptom the pattern exists for.
- The states have genuinely different behaviour, not just different data.
- Transitions are a real part of the domain and worth naming — a pull request's lifecycle, a
  connection's handshake, a grant's validity.

## How to use it

Let each state own its **transitions**, and make the state object immutable so it carries
behaviour rather than mutable data.

```go
// RIGHT — each state is an immutable value that returns the next state, or refuses.
type GrantState interface {
	Use(env Env) (GrantState, error)
}

type Pending struct{}
type Active struct{ Since time.Time }
type Expired struct{}

var ErrGrantExpired = errors.New("grant expired")

func (Pending) Use(env Env) (GrantState, error) { return Active{Since: env.Now()}, nil }

func (a Active) Use(env Env) (GrantState, error) {
	if env.Now().After(a.Since.Add(env.TTL)) {
		return Expired{}, ErrGrantExpired
	}
	return a, nil
}

func (e Expired) Use(Env) (GrantState, error) { return e, ErrGrantExpired }
```

Using an expired grant is refused with `ErrGrantExpired`; the terminal state never answers an
illegal event as if it had succeeded.

The invariant: **every state handles every event in the interface**, even if the handling is
an explicit refusal. A state that cannot answer an event means the interface spans two
machines.

## How not to use it

- **Leaving the conditional in place as well.** If the context still branches on which state
  it holds, nothing was gained and there are now two representations of the same machine.

```go
// WRONG — the context still asks which state it holds.
if _, isExpired := g.state.(Expired); isExpired {
	g.audit.Record("use of expired grant")
}
g.state, err = g.state.Use(env)
// Symptom: expiry behaviour lives in two places, and the next change updates one.
```

- **Mutable state objects holding domain data.** The data belongs to the context; the state
  object holds behaviour. Mutable state objects reintroduce the question of what is true
  after a transition.

```go
// WRONG — the state object owns domain data and mutates it.
func (a *Active) Use(env Env) (GrantState, error) {
	a.UseCount++
	return a, nil
}
// Symptom: after the move to Expired the use count is gone, and nothing says whether it should be.
```

- **An illegal transition that silently does nothing.** Refusal must be explicit — an error,
  a rejected result — or the machine will appear to work while sitting in the wrong state.

```go
// WRONG — revoking a pending grant returns the same state and no error.
func (p Pending) Revoke(Env) (GrantState, error) { return p, nil }
// Symptom: the caller reports the grant revoked; it activates on first use.
```

- **Using it for two states.** A boolean and a conditional are clearer. The pattern earns its
  place at three or four, or when transitions are themselves complicated.

```go
// WRONG — an interface and two types for what is a boolean.
func (Enabled) Toggle() SchedulerState  { return Disabled{} }
func (Disabled) Toggle() SchedulerState { return Enabled{} }
// Symptom: three declarations and a dispatch where isSchedulerEnabled said everything.
```

- **Transitions scattered between the states and the context.** Pick one owner. Split
  ownership means the machine's shape cannot be read from either place.

```go
// WRONG — Active decides expiry by time; the context decides expiry by count.
g.state, err = g.state.Use(env)
if g.useCount > g.useLimit {
	g.state = Expired{}
}
// Symptom: neither Active nor Grant shows every way a grant expires.
```

- **Where a table would be better.** If transitions are pure data — state plus event yields
  state — a transition table is more readable, exhaustively checkable, and testable as data.
  See the global `cc2-table-driven` standard.

```go
// WRONG — a type and a method per transition that is pure data.
func (Queued) Start() RunState        { return InProgress{} }
func (InProgress) Complete() RunState { return Completed{} }
// Symptom: the check run's lifecycle is spread over methods; a map keyed by state and event holds it in one table.
```

**Trade-off:** the machine's overall shape is no longer visible in one place; you read it by
collecting the transitions out of every state class.

**Related:** `strategy.md` — near-identical structure, different intent, `command.md`
