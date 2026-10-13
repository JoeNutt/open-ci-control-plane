# Coupling

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

The degree to which one piece of code must know about another in order to work. It comes in
forms that differ enormously in cost, from passing a value in a signature to depending on
another module's internal behaviour. **Modularity, cohesion, abstraction and separation of
concerns are not peers of coupling — they are the techniques by which it is managed.** That is
why they matter.

## Why it exists as a concept

Because it is what most directly limits how reliably software can be changed. And because the
naive goal — remove it — is impossible: components must communicate, so coupling is inherent.
The engineering decision is **where it sits, what form it takes, and which direction it
points.**

Having the concept means the question stops being "is this modular?" and becomes "does this
reduce, or better place, the coupling — and at what cost?"

## When to use it

As the test applied to any structural decision. Three questions settle most cases:

1. **Which way does it point?** Toward the more stable thing. Volatile may depend on stable,
   never the reverse.
2. **How visible is it?** Coupling through a declared interface is manageable. Coupling
   through shared global state, an implicit call order, or an assumption about internals is
   not — nothing tells you when it breaks.
3. **How much must a caller know?** If using a module correctly requires knowing how it works
   inside, the abstraction has failed regardless of the interface.

## How to use it

Convert coupling downward through this table — worst first:

| Form | Shape | Why it hurts |
|---|---|---|
| **Semantic / behavioural** | Depending on behaviour rather than contract: call order, a side effect, "it returns sorted" | Compiles, breaks silently, invisible in review |
| **Shared mutable state** | A global, a shared cache, a singleton | No owner, action at a distance |
| **Temporal** | A must run before B, unenforced | Works until someone reorders |
| **Structural / content** | Reaching into internals, `a.B().C()` | Binds you to a structure two modules away |
| **Control** | A flag telling the callee which branch to take | The caller is writing the callee's logic |
| **Data** | Passing the values needed, via a signature | The acceptable kind |

```go
// RIGHT — temporal coupling made structural. The type system now enforces the order.
type Raw struct{ bytes []byte }
type Bounded struct{ files []ChangedFile }    // can only exist after bounds passed

func CheckBounds(r Raw, l Limits) (Bounded, error) { /* ... */ }
func Evaluate(b Bounded) Verdict               { /* cannot be called with Raw */ }
```

Also worth tracking, **afferent and efferent**: what depends on this module (high = rigid,
changing it breaks many things) and what it depends on (high = fragile, many things can break
it). Both high is the worst case and is usually a `common`/`shared`/`utils` bucket. Stable
things should be abstract; volatile things concrete.

**A second lens — coupling by consequence** (Nygard), which is often more actionable at system
scale and invisible in a form-based analysis:

- **Developmental** — you cannot release your change until I finish mine.
- **Operational** — my service cannot start unless yours is running.

Ask of any boundary: *can these be built, tested, released and run independently?* Where the
answer is no, name which kind is stopping it.

**The invariant: semantic coupling is made explicit or removed.** If B genuinely requires A to
have run, say so in a type or a signature — never in a comment.

## How not to use it

**Depending on undeclared behaviour.**

```go
// WRONG — relies on Evaluate() returning reasons already sorted.
v := Evaluate(change)
return v.Reasons[0]                // "the most severe one"
// Symptom: compiles, passes today, breaks silently the day Evaluate stops sorting.
// Nothing in the signature promised an order.
```

**Control coupling — a flag selecting the callee's branch.**

```go
// WRONG — the caller is writing the callee's logic.
func Evaluate(c Change, security bool, quality bool, dryRun bool) Verdict
// Symptom: three booleans mean eight behaviours behind one name, and the call site
// reads `Evaluate(c, true, false, true)`. These are separate operations.
```

**Treating loose coupling as an absolute.** It is a preference, and absolutism is itself a
design error:

- **Decoupling usually means more code** — an interface, an adapter, a translation layer that
  does no business work. That is a real cost paid for real benefit. The reflex that "less code
  is good" is wrong at a boundary.
- **Coupling can be too loose.** Indirection past the point of value produces systems nobody
  can trace, and is not free at runtime. "We followed best practice" is not a defence for a
  design nobody can follow.

**Fixing a coupling problem by adding a network call.** Splitting a tangled module into two
services converts a compile-time error into a runtime failure and adds latency, partial
failure and versioning. See `module-boundaries.md`.

**Trade-off:** every reduction in coupling costs code, indirection and often runtime. The
judgement is which boundaries deserve it.

**Related:** `cohesion.md`, `module-boundaries.md`, `dry-across-boundaries.md`
