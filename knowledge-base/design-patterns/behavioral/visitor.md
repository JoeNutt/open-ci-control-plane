# Visitor

*Behavioral · GoF · Authored for this project*

## What it is

An operation over a structure of differing node types is moved into a separate visitor object
with one method per node type. Each node implements `accept`, which calls back the method for
its own type — double dispatch. New operations are added by writing a new visitor, without
changing any node.

## When to use it

- The **node types are stable** and the **operations change often**. That asymmetry is the
  entire justification.
- Several unrelated operations traverse the same structure — evaluate, type-check, pretty-print,
  collect dependencies over one syntax tree.
- Keeping those operations inside the node classes would mix unrelated concerns into every
  node.

## How to use it

Keep traversal order in one place — either every visitor or the structure, consistently — and
make the visitor interface exhaustive.

```typescript
interface Visitor<T> {
  visitRule(n: Rule): T;
  visitRuleSet(n: RuleSet): T;
}

class RuleSet implements Node {
  accept<T>(v: Visitor<T>): T { return v.visitRuleSet(this); }   // double dispatch
}

class CountRules implements Visitor<number> {
  visitRule(_: Rule) { return 1; }
  visitRuleSet(n: RuleSet) { return n.children.reduce((s, c) => s + c.accept(this), 0); }
}
```

The invariant: **adding a node type must break every visitor at compile time.** If it does
not — because the interface has a default case or the language permits partial implementation
— the pattern's one safety property is gone and new node types are silently unhandled.

## How not to use it

- **When node types change more often than operations.** This is the pattern's acknowledged
  weakness, and applying it against the grain means every new node edits every visitor. If
  that is your direction, put the methods on the nodes.
- **A default or catch-all case in the visitor interface.** It converts a compile error into
  a silent omission, which is precisely the failure the pattern prevents.
- **Visitors that need node internals the interface does not expose**, prompting getters that
  open up the node's state to everyone. The encapsulation cost is real and should be weighed
  before adopting.
- **Inconsistent traversal.** If some visitors recurse and the structure also offers traversal,
  nodes get visited twice or not at all. Decide once, where it is documented.
- **In a language with pattern matching over sum types.** An exhaustive `match` or `switch`
  over a sealed hierarchy gives the same compile-time exhaustiveness with none of the double
  dispatch — the pattern is largely a workaround for languages that lack it.
- **Stateful visitors reused across traversals** without a reset. Accumulated state from the
  previous run produces wrong answers on the second.

**Trade-off:** node types become hard to add and operations easy — the exact inverse of
putting methods on the nodes. Choose on which of the two actually changes.

**Related:** `../structural/composite.md`, `interpreter.md`, `iterator.md`
