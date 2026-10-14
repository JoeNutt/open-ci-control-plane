# Engineering practice

How code gets written. How the system is *shaped* is `../architecture/`.

Every entry here is **derived from a global skill and binding on this project**. The skill
remains the authority; if an entry disagrees with its skill, the skill wins and the entry is
the bug.

## The cycle

| Entry | From |
|---|---|
| `tdd-red-green-refactor.md` | `tdd` — and the refactor step is not optional |

## Construction — Clean Code and Code Complete

| Entry | Read it for |
|---|---|
| `naming.md` | Names as the primary documentation; the predicate rule for booleans |
| `routines.md` | One job, the parameter limit and when it bends, no boolean selectors |
| `control-flow.md` | Nominal path first, guard clauses, the three-level nesting limit |
| `classes-and-abstractions.md` | Responsibilities over size, cohesion by actor, information hiding |
| `data-and-scope.md` | Declare at first use, narrow scope, global mutable state |
| `comments.md` | Default to none, the two permitted kinds, the self-justification trap |
| `code-layout.md` | The newspaper metaphor, the stepdown rule, one level of abstraction |
| `duplication.md` | Knowledge duplication, not text duplication |
| `table-driven-methods.md` | When a conditional chain is data wearing a costume |
| `optimisation.md` | Measuring before and after, and the two ways cost intuition fails |

## Failure handling

| Entry | Read it for |
|---|---|
| `defensive-design.md` | The barricade: validate once, at a known boundary, convert to types |
| `errors-and-assertions.md` | Whose fault it is, and why a failure is never a plausible value |

Security is a separate area: `../security/`.
