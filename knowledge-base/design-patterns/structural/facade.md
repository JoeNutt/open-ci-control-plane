# Facade

*Structural · GoF · Authored for this project*

## What it is

A single object provides a simplified interface to a set of interfaces in a subsystem. It
knows which subsystem objects to call, in what order, and with what defaults, and exposes
one operation that means something to the caller. The subsystem remains fully accessible to
anyone who needs more.

## When to use it

- A common path through a complicated subsystem is reproduced by every caller, slightly
  differently, which is duplicated knowledge of the subsystem's sequencing.
- A subsystem's interface is correct but low-level, and most callers want one named outcome
  rather than nine primitives.
- A layer boundary needs a deliberately narrow entry point so what callers depend on is
  visible and small.

## How to use it

Name the facade for the **outcome**, not the subsystem, and let it hold the sequencing
knowledge and nothing else.

```python
class ChangeIngest:                      # named for what it produces
    def __init__(self, forge, extractors, bounds):
        self._forge, self._extractors, self._bounds = forge, extractors, bounds

    def build_document(self, event: ForgeEvent) -> ChangeDocument:
        files = self._forge.list_files(event.pr)
        self._bounds.check(files)                       # order matters, and lives here
        imports = self._extractors.extract(files)
        return assemble(event, files, imports)
```

The invariant: **the facade adds no rules of its own.** It sequences and defaults. The moment
it decides something the subsystem does not, it is a new component and should be named as one.

## How not to use it

- **Letting it accumulate until it is the subsystem.** The usual failure: every new caller
  adds a method, and the facade becomes a god object that knows everything and is changed by
  every feature. Watch for methods whose field usage does not overlap — that is two facades.
- **Hiding the subsystem rather than simplifying it.** If the facade is the only legal way in,
  it must cover every case, and it will grow until it does. Leave the subsystem reachable.
- **Pass-through methods.** A facade method that forwards one call unchanged adds a name and
  a hop. Either it sequences something or it should not exist.
- **Putting a facade over a subsystem you own and could simplify directly.** The pattern
  earns its place against complexity you cannot change; against your own code it is often
  cheaper to fix the interface.
- **Confusing it with Adapter.** Adapter satisfies an interface the client already requires;
  Facade invents a more convenient one.

**Trade-off:** one more place to change when the subsystem changes, and a tendency to grow
that must be actively resisted.

**Related:** `adapter.md`, `proxy.md`
