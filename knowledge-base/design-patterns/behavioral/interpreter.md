# Interpreter

*Behavioral · GoF · Authored for this project*

## What it is

A grammar is represented as a class hierarchy, one type per rule, and evaluation is a method
on those types. A sentence in the language becomes an abstract syntax tree of these objects,
and interpreting it means calling `interpret` on the root and letting each node evaluate its
children.

## Why it exists

When a small language's sentences are handled as strings, every evaluation re-parses them and
every operation over them — evaluate, validate, print — re-implements the grammar ad hoc,
each slightly differently. Representing each grammar rule as a type puts the grammar in one
place, and makes a sentence a tree that can be built, inspected and evaluated without parsing
it again.

## When to use it

- You **own a small language** whose grammar is simple and stable — a filter expression, a
  routing rule, a query fragment, a calculator.
- The trees are evaluated often enough to matter but the grammar changes rarely.
- Representing the grammar as types buys something concrete: type-checked construction,
  exhaustive handling, trees you can inspect and transform rather than strings you re-parse.

## How to use it

Keep each node's `interpret` to **its own rule**, pass evaluation context explicitly, and
bound the recursion. In Go the hierarchy is an interface with one struct type per rule.

```go
// RIGHT — one type per grammar rule, each evaluating only itself and its children.
type Expr interface {
	Eval(env Env) bool
}

type And struct{ Left, Right Expr }

func (a And) Eval(env Env) bool { return a.Left.Eval(env) && a.Right.Eval(env) }

type PathMatches struct{ Glob string }

func (p PathMatches) Eval(env Env) bool { return env.MatchPath(p.Glob) }
```

The invariant: **a node evaluates only itself and its children.** A node reaching the root,
a sibling or global state has made the tree's meaning depend on its position.

## How not to use it

- **For anything but a small, stable grammar.** The class-per-rule cost grows with the
  grammar, and a real language wants a parser generator and a separate evaluator, not one
  class per production.

```go
// WRONG — a type per production, for a language that is still growing.
type Let struct{ Binding Binding }
type Lambda struct{ Fn Function }
type Call struct{ Args []Expr }
type Comprehension struct{ Clauses []Clause }
// Symptom: each release adds node types, and evaluation is spread across all of them.
```

- **Unbounded recursion over caller-supplied trees.** A deeply nested expression is a stack
  overflow, and if the expression comes from outside the process it is a denial-of-service
  vector. Bound depth at parse time, before evaluating.

```go
// WRONG — the tree came from a request body, and nothing bounded its depth.
expr, err := parseExpr(body)
if err != nil {
	return err
}
isAllowed := expr.Eval(env)
// Symptom: a body of a million nested terms exceeds the goroutine stack limit, a fatal error no recover catches.
```

- **Mixing parsing into the nodes.** This pattern covers evaluation. A node that also knows
  how to parse itself from text couples the grammar's syntax to its semantics, and changing
  the syntax then touches every rule.

```go
// WRONG — the node knows its own text syntax as well as its meaning.
func (a *And) UnmarshalText(text []byte) error {
	left, right, isFound := bytes.Cut(text, []byte(" && "))
	if !isFound {
		return errNotAnd
	}
	return a.parseOperands(left, right)
}
// Symptom: changing && to "and" in the syntax edits every node type.
```

- **Reaching for it when a lookup table or a closure would do.** A fixed set of named
  conditions is a map of predicates. The pattern earns its place when expressions **compose**.

```go
// WRONG — node types for conditions that never compose.
type IsDraft struct{}
type IsFromFork struct{}

func (IsDraft) Eval(env Env) bool    { return env.PR.Draft }
func (IsFromFork) Eval(env Env) bool { return env.PR.FromFork }
// Symptom: no node has children; a map[string]func(PullRequest) bool says the same.
```

- **Writing your own when a real engine exists.** For this project specifically: policy
  expressions are OPA's job, not a hand-rolled interpreter's. See
  `../../technology/rego-overview.md`.

```go
// WRONG — a hand-rolled policy expression beside an embedded OPA.
type MinApprovals struct{ Count int }

func (m MinApprovals) Eval(env Env) bool { return len(env.PR.Approvals) >= m.Count }
// Symptom: two policy languages with two semantics, and only one of them is Rego.
```

**Trade-off:** the most class-heavy pattern in the catalogue relative to the work it does, and
the one most often better replaced by an existing engine.

**Related:** `visitor.md` — the usual companion for adding operations over the same tree,
`../structural/composite.md`
