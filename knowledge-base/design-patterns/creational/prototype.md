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

Go has no classes, so no `clone` is inherited down a hierarchy: each type declares its own
`Clone`, and where the concrete type is unknown the caller holds an interface whose method
set includes it. Struct assignment, Go's only built-in copy, is shallow.

```go
// RIGHT — Clone states which state is owned and which is shared.
func (w *Workflow) Clone() *Workflow {
    steps := make([]*Step, len(w.Steps))
    for i, s := range w.Steps {
        steps[i] = s.Clone() // deep: steps are owned
    }
    return &Workflow{Name: w.Name, Steps: steps, Registry: w.Registry} // shallow: registry is shared
}
```

The invariant: **after cloning, mutating the copy must not change the original** for every
piece of state the copy is said to own. Anything deliberately shared is named in the
contract.

## How not to use it

- **Relying on a language's default copy.** Python's `copy.copy`, Go's struct assignment and
  C++'s implicit copy constructor all make a shallow copy, which silently shares slices,
  maps, pointers and handles. The resulting aliasing bug appears far from the clone.

```go
// WRONG — struct assignment copies the slice header, not the steps.
copied := *original
copied.Steps[0].Name = "lint"
// Symptom: original.Steps[0].Name is now "lint" too, found wherever the original is next read.
```

- **Cloning objects that own external resources.** An open file, a socket, a lock or a
  transaction cannot be meaningfully copied. Two objects believing they own one handle is
  worse than an expensive constructor.

```go
// WRONG — both copies believe they own the file.
func (s *Spool) Clone() *Spool { return &Spool{out: s.out} }
// Symptom: one copy's Close makes the other's next Write fail with os.ErrClosed.
```

- **Using it to dodge an awkward constructor.** If construction is painful, fix construction
  — `builder.md` exists for exactly that. Cloning to avoid naming parameters hides the
  problem and adds an aliasing hazard.

```go
// WRONG — a seed instance cloned so no call site has to state its fields.
check := defaultCheck.Clone()
check.Name = "lint"
// Symptom: every unset field is whatever defaultCheck held, and nothing says which those are.
```

- **Treating JavaScript's prototype chain as this pattern.** They share a name and an
  intuition, not a mechanism: the chain is delegation at lookup time, not copying at creation
  time. Go's nearest equivalent is embedding a pointer, which delegates in the same way.

```go
// WRONG — embedding the prototype delegates to it; nothing was copied.
type LintWorkflow struct{ *Workflow }

lint := LintWorkflow{Workflow: base}
lint.Name = "lint"
// Symptom: base.Name is now "lint".
```

**Trade-off:** every class in the hierarchy owes a correct clone, and a new field added
without updating `clone` is a silent aliasing defect that no compiler catches.

**Related:** `builder.md`, `factory-method.md`, `../structural/flyweight.md`
