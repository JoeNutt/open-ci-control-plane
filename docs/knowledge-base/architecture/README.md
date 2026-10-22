# Architecture

The decisions that are expensive to reverse: what depends on what, and where the boundaries
sit. How a single routine is written is construction, not architecture.

Every entry is *derived from the global `architecture` skill* and its references, and is
**binding on this project**. The skill remains the authority; if an entry here disagrees with
it, the skill wins and the entry is the bug.

| Entry | Read it for |
|---|---|
| `dependency-direction.md` | The one rule. Source dependencies point inward, toward policy |
| `ports-and-adapters.md` | The same rule in usable terms, and who owns the interface |
| `layering.md` | Where a given piece of code goes, and what belongs in each layer |
| `coupling.md` | Forms of coupling worst-first, direction, visibility, and that it cannot be removed |
| `cohesion.md` | Things that change together live together, and why layer-first folders break it |
| `module-boundaries.md` | Where to draw a boundary, and when a module should become a process |
| `testability-as-design-feedback.md` | Hard-to-write tests as a design signal, not a tooling problem |
| `deployability.md` | Release cadence as an architectural property |
| `proportionality.md` | Scaling structure to what the system actually is |
| `incremental-architecture.md` | Reversible decisions, deferring choices, structural change as experiment |
| `dry-across-boundaries.md` | Deduplicate within a boundary, translate across one |

**Separation of concerns** and **information hiding** are not separate entries: they are
techniques for managing coupling, and are covered in `coupling.md` and `cohesion.md` where
they are actually applied. The same is true of **modularity**.
