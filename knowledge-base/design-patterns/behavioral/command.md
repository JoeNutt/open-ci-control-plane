# Command

*Behavioral · GoF · Authored for this project*

## What it is

A request is turned into an object carrying the operation and its parameters, with a single
method to execute it. Because the request is now a value, it can be stored, passed, queued,
logged, replayed and — if it also records how to reverse itself — undone.

## When to use it

- Operations must be **deferred or scheduled**: a job queue, a retry buffer, a transaction
  log.
- Undo or redo is required, and each operation can state its inverse.
- Operations must be recorded for audit or replay, which is the same requirement as an event
  log.
- The invoker must stay ignorant of what it is invoking — a toolbar button, a keybinding, a
  worker pulling from a channel.

## How to use it

Make the command a **value** that carries everything it needs, so it survives serialisation
and a process restart.

```go
// RIGHT — every input is a field, fixed when the command is built.
type Command interface {
    Execute(ctx context.Context, env Env) error
}

type RevokeGrant struct {
    GrantID string
    Actor   string
}

func (c RevokeGrant) Execute(ctx context.Context, env Env) error {
    return env.Grants.Revoke(ctx, c.GrantID, c.Actor)
}
```

The invariant: **a command captures its inputs at construction, not at execution.** If it
reads mutable shared state when it runs, the recorded command and the executed command are
different operations, and replay produces a different result.

## How not to use it

- **Holding a live reference to mutable state.** A command that resolves its target when it
  executes cannot be replayed, queued or reasoned about. It is a closure over the present.

```go
// WRONG — the target is read from the session when the command runs.
type RevokeGrant struct{ session *Session }

func (c RevokeGrant) Execute(ctx context.Context, env Env) error {
    return env.Grants.Revoke(ctx, c.session.CurrentGrantID, c.session.User)
}
// Symptom: replaying the log revokes whichever grant the session holds now.
```

- **Undo that is not actually inverse.** Storing "the old value" and restoring it is wrong
  the moment something else changed in between. Either make undo genuinely inverse or make
  the operation a new forward command that compensates.

```go
// WRONG — undo restores a snapshot instead of reversing the change.
func (c SetThreshold) Undo(ctx context.Context, env Env) error {
    return env.Thresholds.Set(ctx, c.Policy, c.Previous)
}
// Symptom: undoing after another operator's edit silently discards their edit too.
```

- **A command per method on one object.** That is a hand-written vtable. The pattern pays
  when commands are stored, queued or undone — not when they are invoked immediately at the
  point of construction.

```go
// WRONG — a command type per method, executed where it is built.
type OpenCheckRun struct{ ID string }
type CloseCheckRun struct{ ID string }

err := OpenCheckRun{ID: id}.Execute(ctx, env)
// Symptom: nothing stores, queues or undoes these; checkRuns.Open(ctx, id) said the same.
```

- **Putting business rules inside execute.** A command that decides *whether* it is allowed
  mixes authorisation with the operation and cannot be validated before being queued. Decide
  at enqueue; execute does.

```go
// WRONG — the command decides whether it is permitted only when it runs.
func (c RevokeGrant) Execute(ctx context.Context, env Env) error {
    if !env.Authz.CanRevoke(c.Actor, c.GrantID) {
        return ErrForbidden
    }
    return env.Grants.Revoke(ctx, c.GrantID, c.Actor)
}
// Symptom: a forbidden command is accepted into the queue and fails hours later, far from the request.
```

- **In a language with first-class functions, using a class where a closure fits.** If
  nothing needs the command to be serialisable or inspectable, a function is the pattern.

```go
// WRONG — a type and a method for an operation nothing stores or inspects.
type notifyCommand struct{ message string }

func (c notifyCommand) Execute(ctx context.Context, env Env) error {
    return env.Notify(ctx, c.message)
}
// Symptom: a declaration per operation where a func(context.Context, Env) error was the whole pattern.
```

**Trade-off:** one type per operation, and a serialisation format that becomes a
compatibility contract the moment commands outlive a deployment.

**Related:** `memento.md`, `chain-of-responsibility.md`, `strategy.md`
