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

```typescript
interface Command { execute(ctx: Context): Promise<void> }

class RevokeGrant implements Command {
  constructor(private readonly grantId: string, private readonly actor: string) {}
  async execute(ctx: Context) { await ctx.grants.revoke(this.grantId, this.actor); }
}
```

The invariant: **a command captures its inputs at construction, not at execution.** If it
reads mutable shared state when it runs, the recorded command and the executed command are
different operations, and replay produces a different result.

## How not to use it

- **Holding a live reference to mutable state.** A command that resolves its target when it
  executes cannot be replayed, queued or reasoned about. It is a closure over the present.
- **Undo that is not actually inverse.** Storing "the old value" and restoring it is wrong
  the moment something else changed in between. Either make undo genuinely inverse or make
  the operation a new forward command that compensates.
- **A command per method on one object.** That is a hand-written vtable. The pattern pays
  when commands are stored, queued or undone — not when they are invoked immediately at the
  point of construction.
- **Putting business rules inside execute.** A command that decides *whether* it is allowed
  mixes authorisation with the operation and cannot be validated before being queued. Decide
  at enqueue; execute does.
- **In a language with first-class functions, using a class where a closure fits.** If
  nothing needs the command to be serialisable or inspectable, a function is the pattern.

**Trade-off:** one type per operation, and a serialisation format that becomes a
compatibility contract the moment commands outlive a deployment.

**Related:** `memento.md`, `chain-of-responsibility.md`, `strategy.md`
