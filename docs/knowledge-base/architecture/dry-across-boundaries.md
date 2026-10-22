# Deduplicate within a boundary, translate across one

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

A qualification on "don't repeat yourself". A single canonical representation of each piece of
knowledge is right **within** a function, a module, a service, or roughly one repository and
deployment pipeline. Applied **across** independently developed modules or services it
inverts: enforcing one shared representation couples them together, and that coupling usually
costs more than the duplication did.

## Why it exists

Because DRY as usually stated is too simplistic, and the over-application is expensive and
common. Two services each with their own notion of "customer", translated at the boundary, are
often better designed than two services sharing one model — because the shared model makes
every change to either one a negotiation with the other.

It also exists to protect a specific, correct duplication: the one between a database row, a
domain entity and an API response. That looks like three copies of the same thing and is not.
They change for different reasons, and each is the right shape for its own side.

## When to use it

Use the boundary as the test whenever you spot duplication:

- **Inside one module?** Remove it. Exact duplication created in the green step of a TDD cycle
  goes now, not on its third appearance.
- **Across a module or service boundary?** That is an architecture question, not a refactor.
  Translate rather than share, unless the two sides genuinely have one owner and one release.

The distinction also decides where a *rule* lives. A limit expressed in the domain and again
in a handler is knowledge duplication inside a boundary — give it one owner. A limit the
domain enforces and a client also checks for a better error message is translation across one.

## How to use it

Let each side own its shape, and pay the mapping tax deliberately.

```go
// RIGHT — three representations, three reasons to change, mapped at the edges.
// persistence: shaped by the schema
type changeRow struct {
    ID        int64     `db:"id"`
    PathsJSON []byte    `db:"paths"`
    CreatedAt time.Time `db:"created_at"`
}

// domain: shaped by the rules
type Change struct {
    Paths []Path
    Kinds KindSet
}

// wire: shaped by the published contract, versioned independently
type changeDTO struct {
    SchemaVersion int      `json:"schema_version"`
    Paths         []string `json:"paths"`
}

func rowToDomain(r changeRow) (Change, error) { /* the tax, paid on purpose */ }
```

```go
// RIGHT — one owner for a rule inside a boundary.
package bounds
const MaxFiles = 1000                   // declared once
// the handler and the domain both read bounds.MaxFiles; neither restates 1000
```

**The invariant: knowledge has exactly one owner inside a boundary, and no owner across
one.** A comment saying "keep this in sync with X" is an admission that the knowledge has no
owner — give it one, or accept the duplication deliberately because X is across a boundary.

## How not to use it

**A shared model across a boundary.**

```go
// WRONG — one package imported by two independently released services.
package sharedtypes          // imported by policy-service and matrix-service

type Change struct {
    Paths []string
    Kinds []string
    Plan  *TestPlan          // matrix needs this; policy does not
}
// Symptom: adding a field for one service requires releasing both. The services
// are decoupled in code and developmentally coupled by the shared type.
```

**Eliminating the mapping tax by using one struct everywhere.**

```go
// WRONG — the database row IS the domain type IS the API response.
type Change struct {
    ID        int64     `db:"id"        json:"id"`
    PathsJSON []byte    `db:"paths"     json:"-"`
    Paths     []string  `db:"-"         json:"paths"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}
// Symptom: a column rename is an API break, and the struct carries fields that are
// meaningless on two of its three sides. The tags are the tell.
```

**Treating the qualification as permission to duplicate inside a module.** It is not. Within a
boundary the rule stands in full, and the same literal in two files is the strongest form of
the defect.

**Deduplicating by inventing a `shared` package.** That is not removing duplication, it is
relocating it into a module with no cohesion and dependencies in every direction. See
`cohesion.md`.

**Trade-off:** translation means mapping code, and mapping code is where silent field-drop bugs
live. It is bought in exchange for each side being able to change without the other's consent.

**Related:** `coupling.md`, `module-boundaries.md`, `layering.md`
