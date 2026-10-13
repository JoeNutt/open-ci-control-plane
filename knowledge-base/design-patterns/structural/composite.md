# Composite

*Structural · GoF · Authored for this project*

## What it is

Individual objects and groups of objects implement the same interface, so a client treats a
leaf and a whole subtree identically. A composite node holds children — leaves or further
composites — and implements the shared operations by delegating to them and combining the
results.

## When to use it

- The data genuinely forms a **part-whole hierarchy**: a filesystem, an expression tree, a
  policy set containing policy sets, a UI layout.
- Clients should be able to ignore the difference between one element and many — a renderer
  that draws a widget or a panel of widgets with the same call.
- An operation is naturally recursive over the structure: total size, validate, render,
  evaluate.

## How to use it

Put only the operations that are meaningful for **both** leaf and composite on the shared
interface, and keep child management on the composite type.

```typescript
interface Node { evaluate(input: Input): Verdict }

class Rule implements Node {                      // leaf
  evaluate(input: Input): Verdict { /* ... */ }
}

class RuleSet implements Node {                   // composite
  private children: Node[] = [];
  add(child: Node) { this.children.push(child); }  // child management lives here only
  evaluate(input: Input): Verdict {
    return combine(this.children.map(c => c.evaluate(input)));
  }
}
```

The invariant: **the shared interface contains no operation a leaf must refuse.** The moment
it does, clients start testing which kind they hold and the uniformity is gone.

## How not to use it

- **Putting `add` and `remove` on the shared interface.** GoF discusses this openly as a
  trade between transparency and safety, and the common resolution is to accept the cast: a
  leaf forced to implement `add` either throws — a runtime error the type system could have
  prevented — or silently does nothing, which is worse.
- **Forcing a flat collection into a tree.** If there is no genuine part-whole relationship,
  a list is a list. The pattern's cost is recursion everywhere, paid for nothing.
- **Ignoring cycles.** Nothing in the structure prevents a composite containing an ancestor,
  and the symptom is a stack overflow during a traversal, far from the code that built the
  cycle. If the structure is caller-assembled, check on insert.
- **Unbounded depth on untrusted input.** A recursive evaluate over a caller-supplied tree is
  a denial-of-service vector. Bound the depth before traversing, not during.
- **Caching aggregates without invalidation.** A composite that memoises its total and does
  not hear about a child's change returns a confidently wrong answer.

**Trade-off:** the type system stops distinguishing one from many, which is the benefit and
also the reason a leaf-only mistake surfaces at runtime.

**Related:** `decorator.md`, `../behavioral/visitor.md`, `../behavioral/iterator.md`
