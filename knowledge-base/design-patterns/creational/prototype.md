# Prototype

*Creational · GoF · Authored for this project*

## What it is

New objects are produced by copying an existing instance rather than by calling a
constructor. The instance being copied — the prototype — carries the configuration the copy
should start from, and the copying operation lives on the object itself, so the caller
creates a new object of the right class without naming that class.

## When to use it

- Constructing from scratch is genuinely expensive and an already-initialised instance can
  seed the copy — a parsed template, a warmed index, a connection-less client configured from
  slow-loading settings.
- The concrete class is not known at the call site but an instance of it is in hand.
- A system needs many near-identical objects differing in a few fields, registered as named
  prototypes and cloned on demand.

## How to use it

Decide **explicitly** whether the copy is shallow or deep, and make that decision part of the
documented contract rather than an accident of the language's copy semantics.

```python
class Workflow:
    def clone(self) -> "Workflow":
        copied = Workflow(self.name)
        copied.steps = [s.clone() for s in self.steps]   # deep: steps are owned
        copied.registry = self.registry                   # shallow: registry is shared
        return copied
```

The invariant: **after cloning, mutating the copy must not change the original** for every
piece of state the copy is said to own. Anything deliberately shared is named in the
contract.

## How not to use it

- **Relying on a language's default copy.** Python's `copy.copy`, Go's struct assignment and
  C++'s implicit copy constructor all make a shallow copy, which silently shares slices,
  maps, pointers and handles. The resulting aliasing bug appears far from the clone.
- **Cloning objects that own external resources.** An open file, a socket, a lock or a
  transaction cannot be meaningfully copied. Two objects believing they own one handle is
  worse than an expensive constructor.
- **Using it to dodge an awkward constructor.** If construction is painful, fix construction
  — `builder.md` exists for exactly that. Cloning to avoid naming parameters hides the
  problem and adds an aliasing hazard.
- **Treating JavaScript's prototype chain as this pattern.** They share a name and an
  intuition, not a mechanism: the chain is delegation at lookup time, not copying at creation
  time.

**Trade-off:** every class in the hierarchy owes a correct clone, and a new field added
without updating `clone` is a silent aliasing defect that no compiler catches.

**Related:** `builder.md`, `factory-method.md`, `../structural/flyweight.md`
