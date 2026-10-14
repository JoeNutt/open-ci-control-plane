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

```go
// RIGHT — Rule and RuleSet both satisfy Node; only RuleSet manages children.
type Node interface {
    Evaluate(in Input) Verdict
}

type Rule struct{ Query string }

func (r Rule) Evaluate(in Input) Verdict { /* ... */ }

type RuleSet struct{ children []Node }

func (s *RuleSet) Add(child Node) { s.children = append(s.children, child) }

func (s *RuleSet) Evaluate(in Input) Verdict {
    verdicts := make([]Verdict, 0, len(s.children))
    for _, c := range s.children {
        verdicts = append(verdicts, c.Evaluate(in))
    }
    return combine(verdicts)
}
```

The invariant: **the shared interface contains no operation a leaf must refuse.** The moment
it does, clients start testing which kind they hold and the uniformity is gone.

## How not to use it

- **Putting `add` and `remove` on the shared interface.** GoF discusses this openly as a
  trade between transparency and safety, and the common resolution is to accept the cast: a
  leaf forced to implement `add` either throws — a runtime error the type system could have
  prevented — or silently does nothing, which is worse. In Go the cast is a type assertion to
  `*RuleSet`.

```go
// WRONG — child management on the shared interface, so a leaf must refuse it.
type Node interface {
    Evaluate(in Input) Verdict
    Add(child Node) error
}

func (Rule) Add(Node) error { return errLeafHasNoChildren }
// Symptom: every caller handles an error only a leaf returns; a leaf returning nil drops the child.
```

- **Forcing a flat collection into a tree.** If there is no genuine part-whole relationship,
  a list is a list. The pattern's cost is recursion everywhere, paid for nothing.

```go
// WRONG — a composite for checks that never nest.
type CheckGroup struct{ children []CheckNode }

func (g CheckGroup) Run(ctx context.Context) Result { /* recurse into children */ }
// Symptom: every group holds only leaves; a []Check and one loop did the job.
```

- **Ignoring cycles.** Nothing in the structure prevents a composite containing an ancestor,
  and the symptom is a stack overflow during a traversal, far from the code that built the
  cycle. If the structure is caller-assembled, check on insert.

```go
// WRONG — a caller-assembled tree, and Add checks nothing.
parent.Add(child)
child.Add(parent)
// Symptom: the next Evaluate exceeds the goroutine stack limit, far from these two lines.
```

- **Unbounded depth on untrusted input.** A recursive evaluate over a caller-supplied tree is
  a denial-of-service vector. Bound the depth before traversing, not during.

```go
// WRONG — a tenant's uploaded policy set, evaluated without a depth check.
set, err := decodeRuleSet(upload)
if err != nil {
    return err
}
verdict := set.Evaluate(in)
// Symptom: one deeply nested upload is a fatal stack overflow for every tenant on the process.
```

- **Caching aggregates without invalidation.** A composite that memoises its total and does
  not hear about a child's change returns a confidently wrong answer.

```go
// WRONG — the count is memoised, and no child can tell its parent it changed.
func (s *RuleSet) RuleCount() int {
    if !s.isCountCached {
        s.cachedCount, s.isCountCached = s.countChildren(), true
    }
    return s.cachedCount
}
// Symptom: after a nested set gains a rule, RuleCount still returns the old total.
```

**Trade-off:** the type system stops distinguishing one from many, which is the benefit and
also the reason a leaf-only mistake surfaces at runtime.

**Related:** `decorator.md`, `../behavioral/visitor.md`, `../behavioral/iterator.md`
