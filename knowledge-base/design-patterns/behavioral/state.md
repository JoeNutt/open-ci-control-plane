# State

*Behavioral · GoF · Authored for this project*

## What it is

An object's behaviour is delegated to a state object, and changing state means swapping that
object. Each state class implements the same interface and handles events in its own way,
including deciding which state comes next. The conditional over a state field disappears into
polymorphism.

## When to use it

- A **conditional over the same state field appears in several methods**, and adding a state
  means finding and editing all of them. This is the symptom the pattern exists for.
- The states have genuinely different behaviour, not just different data.
- Transitions are a real part of the domain and worth naming — a pull request's lifecycle, a
  connection's handshake, a grant's validity.

## How to use it

Let each state own its **transitions**, and make the state object immutable so it carries
behaviour rather than mutable data.

```kotlin
sealed interface GrantState {
    fun use(ctx: Context): GrantState          // returns the next state
}

object Pending : GrantState {
    override fun use(ctx: Context) = Active(ctx.now())
}

data class Active(val since: Instant) : GrantState {
    override fun use(ctx: Context) =
        if (ctx.now() > since + ctx.ttl) Expired else this
}

object Expired : GrantState {
    override fun use(ctx: Context) = Expired   // terminal, and says so
}
```

The invariant: **every state handles every event in the interface**, even if the handling is
an explicit refusal. A state that cannot answer an event means the interface spans two
machines.

## How not to use it

- **Leaving the conditional in place as well.** If the context still branches on which state
  it holds, nothing was gained and there are now two representations of the same machine.
- **Mutable state objects holding domain data.** The data belongs to the context; the state
  object holds behaviour. Mutable state objects reintroduce the question of what is true
  after a transition.
- **An illegal transition that silently does nothing.** Refusal must be explicit — an error,
  a rejected result — or the machine will appear to work while sitting in the wrong state.
- **Using it for two states.** A boolean and a conditional are clearer. The pattern earns its
  place at three or four, or when transitions are themselves complicated.
- **Transitions scattered between the states and the context.** Pick one owner. Split
  ownership means the machine's shape cannot be read from either place.
- **Where a table would be better.** If transitions are pure data — state plus event yields
  state — a transition table is more readable, exhaustively checkable, and testable as data.
  See the global `cc2-table-driven` standard.

**Trade-off:** the machine's overall shape is no longer visible in one place; you read it by
collecting the transitions out of every state class.

**Related:** `strategy.md` — near-identical structure, different intent, `command.md`
