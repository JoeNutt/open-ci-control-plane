# Interpreter

*Behavioral · GoF · Authored for this project*

## What it is

A grammar is represented as a class hierarchy, one type per rule, and evaluation is a method
on those types. A sentence in the language becomes an abstract syntax tree of these objects,
and interpreting it means calling `interpret` on the root and letting each node evaluate its
children.

## When to use it

- You **own a small language** whose grammar is simple and stable — a filter expression, a
  routing rule, a query fragment, a calculator.
- The trees are evaluated often enough to matter but the grammar changes rarely.
- Representing the grammar as types buys something concrete: type-checked construction,
  exhaustive handling, trees you can inspect and transform rather than strings you re-parse.

## How to use it

Keep each node's `interpret` to **its own rule**, pass evaluation context explicitly, and
bound the recursion.

```python
class Expr:                     # one class per grammar rule
    def interpret(self, ctx: Context) -> Value: ...

class And(Expr):
    def __init__(self, left: Expr, right: Expr): self.left, self.right = left, right
    def interpret(self, ctx):
        return self.left.interpret(ctx) and self.right.interpret(ctx)

class PathMatches(Expr):
    def __init__(self, glob: str): self.glob = glob
    def interpret(self, ctx): return ctx.match(self.glob)
```

The invariant: **a node evaluates only itself and its children.** A node reaching the root,
a sibling or global state has made the tree's meaning depend on its position.

## How not to use it

- **For anything but a small, stable grammar.** The class-per-rule cost grows with the
  grammar, and a real language wants a parser generator and a separate evaluator, not one
  class per production.
- **Unbounded recursion over caller-supplied trees.** A deeply nested expression is a stack
  overflow, and if the expression comes from outside the process it is a denial-of-service
  vector. Bound depth at parse time, before evaluating.
- **Mixing parsing into the nodes.** This pattern covers evaluation. A node that also knows
  how to parse itself from text couples the grammar's syntax to its semantics, and changing
  the syntax then touches every rule.
- **Reaching for it when a lookup table or a closure would do.** A fixed set of named
  conditions is a map of predicates. The pattern earns its place when expressions **compose**.
- **Writing your own when a real engine exists.** For this project specifically: policy
  expressions are OPA's job, not a hand-rolled interpreter's. See
  `../../technology/rego-overview.md`.

**Trade-off:** the most class-heavy pattern in the catalogue relative to the work it does, and
the one most often better replaced by an existing engine.

**Related:** `visitor.md` — the usual companion for adding operations over the same tree,
`../structural/composite.md`
