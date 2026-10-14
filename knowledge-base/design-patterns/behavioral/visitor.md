# Visitor

*Behavioral · GoF · Authored for this project*

## What it is

An operation over a structure of differing node types is moved into a separate visitor object
with one method per node type. Each node implements `accept`, which calls back the method for
its own type — double dispatch. New operations are added by writing a new visitor, without
changing any node.

## Why it exists

Adding an operation over a structure of several node types normally means adding a method to
every node type, so each new operation touches every node and mixes unrelated concerns into
each. Writing the operation outside the nodes as a type switch avoids that but loses any check
that every node type is handled. The visitor puts each operation in one type with a method
per node type, and the interface makes the compiler insist every node type is covered.

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

Go has no method overloading, so each visit method carries the node type in its name, and
methods cannot take type parameters, so `Accept` cannot be generic in its result. A Go visitor
accumulates its answer in its own fields. The interface's method set still gives the safety
property: a visitor missing a method does not satisfy `Visitor` and fails to compile.

```go
// RIGHT — Accept dispatches on the node's own type; the visitor is exhaustive.
type Visitor interface {
	VisitRule(r *Rule)
	VisitRuleSet(s *RuleSet)
}

func (r *Rule) Accept(v Visitor)    { v.VisitRule(r) }
func (s *RuleSet) Accept(v Visitor) { v.VisitRuleSet(s) }

type ruleCounter struct{ count int }

func (c *ruleCounter) VisitRule(*Rule) { c.count++ }

func (c *ruleCounter) VisitRuleSet(s *RuleSet) {
	for _, child := range s.Children {
		child.Accept(c)
	}
}
```

The invariant: **adding a node type must break every visitor at compile time.** If it does
not — because a default implementation is embedded, or the operation is a type switch —
the pattern's one safety property is gone and new node types are silently unhandled.

## How not to use it

- **When node types change more often than operations.** This is the pattern's acknowledged
  weakness, and applying it against the grain means every new node edits every visitor. If
  that is your direction, put the methods on the nodes.

```go
// WRONG — the third node type this quarter, and the operations have not changed.
type Visitor interface {
	VisitRule(*Rule)
	VisitRuleSet(*RuleSet)
	VisitWaiver(*Waiver)
}
// Symptom: each new node edits every visitor in the same change.
```

- **A default or catch-all case in the visitor interface.** It converts a compile error into
  a silent omission, which is precisely the failure the pattern prevents.

```go
// WRONG — an embeddable default makes every visitor complete by promotion.
type BaseVisitor struct{}

func (BaseVisitor) VisitRule(*Rule)       {}
func (BaseVisitor) VisitRuleSet(*RuleSet) {}
func (BaseVisitor) VisitWaiver(*Waiver)   {}
// Symptom: every visitor embedding BaseVisitor compiles, and none of them handles waivers.
```

- **Visitors that need node internals the interface does not expose**, prompting getters that
  open up the node's state to everyone. The encapsulation cost is real and should be weighed
  before adopting.

```go
// WRONG — accessors exported only so one visitor can reach inside.
func (r *Rule) Compiled() compiledRule { return r.compiled }
func (r *Rule) CacheKey() string       { return r.cacheKey }
// Symptom: every package can now read the rule's compiled form and depend on it.
```

- **Inconsistent traversal.** If some visitors recurse and the structure also offers traversal,
  nodes get visited twice or not at all. Decide once, where it is documented.

```go
// WRONG — the structure recurses, and ruleCounter.VisitRuleSet recurses too.
func (s *RuleSet) Accept(v Visitor) {
	v.VisitRuleSet(s)
	for _, child := range s.Children {
		child.Accept(v)
	}
}
// Symptom: every rule is counted twice.
```

- **Replacing the visitor with an unchecked type switch.** A type switch over the node
  interface dispatches with none of the double-dispatch ceremony, but Go's compiler does not
  check it for exhaustiveness, so it gives up the one guarantee the visitor exists to provide.
  If a type switch is used, its `default` case must return an error, never fall through.

```go
// WRONG — a type switch trusted to be exhaustive, with no default.
switch n := node.(type) {
case *Rule:
	count++
case *RuleSet:
	count += countRules(n)
}
// Symptom: a new node type compiles, falls through, and is silently uncounted.
```

- **Stateful visitors reused across traversals** without a reset. Accumulated state from the
  previous run produces wrong answers on the second.

```go
// WRONG — one counter carried across traversals.
counter := &ruleCounter{}
for _, set := range sets {
	set.Accept(counter)
	report(set, counter.count)
}
// Symptom: each set's count includes every set before it.
```

**Trade-off:** node types become hard to add and operations easy — the exact inverse of
putting methods on the nodes. Choose on which of the two actually changes.

**Related:** `../structural/composite.md`, `interpreter.md`, `iterator.md`
