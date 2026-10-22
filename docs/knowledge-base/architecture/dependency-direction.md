# Dependency direction

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

Source code dependencies point only inward, toward higher-level policy. Nothing in an inner
layer may name anything in an outer one — no type, function or variable. The business rules
must not know that a web framework, a database driver or a message queue exists. Control flows
outward at runtime; the *compile-time* dependency never does.

## Why it exists

To keep the decisions that matter independent of the decisions that do not. A rule about how a
pull request is judged outlives three web frameworks and two databases. If the rule imports
the database driver, every infrastructure change becomes a change to the rule, and the rule
cannot be tested without the infrastructure.

Without the direction, "layers" are folders. A `domain/` package importing an SDK is not a
domain layer whatever the directory is called, and nothing in the build will tell you.

## When to use it

Always, in any system with business rules worth protecting. The direction is the
non-negotiable part of layered architecture — the *number* of layers is a judgement call and
`proportionality.md` covers when fewer is right.

The rule is cheapest to apply from the first commit and most expensive to retrofit, because
retrofitting means finding every outward reference that already exists.

## How to use it

The inner layer declares an interface for what it needs; the outer layer implements it;
`main` wires them. `main` is the outermost thing in the system: it knows every concrete type
and nothing knows about it.

```go
// RIGHT — package policy (inner). Declares its need. Imports nothing outward.
package policy

type ChangeSource interface {
    FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error)
}

type Gate struct{ src ChangeSource }

func (g Gate) Decide(ctx context.Context, pr PullRequest) (Decision, error) {
    files, err := g.src.FilesChanged(ctx, pr)
    if err != nil {
        return Decision{}, err
    }
    return evaluate(files), nil
}
```

```go
// RIGHT — package forge (outer). Implements the inner interface.
package forge

type GitHubSource struct{ api *github.Client }

func (s GitHubSource) FilesChanged(ctx context.Context, pr policy.PullRequest) ([]policy.ChangedFile, error) {
    // translation lives here; no github type escapes
}
```

```go
// RIGHT — main. Knows every concrete type. Nothing imports main.
gate := policy.Gate{Src: forge.GitHubSource{API: client}}
```

**The invariant: grep the inner package's imports.** If a framework, driver or vendor SDK
appears there, the rule is broken regardless of folder structure. That check is mechanical,
so automate it — see `layering.md`.

## How not to use it

**The domain importing infrastructure directly.**

```go
// WRONG — package policy (inner) imports the forge SDK.
package policy

import "github.com/google/go-github/v60/github"

type Gate struct{ api *github.Client }

func (g Gate) Decide(ctx context.Context, owner, repo string, n int) (Decision, error) {
    files, _, err := g.api.PullRequests.ListFiles(ctx, owner, repo, n, nil)
    // ...
}
// Symptom: the rule cannot be tested without a GitHub client, and changing forge
// edits the business logic. The import statement is the whole proof.
```

**Defining the port next to its implementation.** This is the most commonly missed version,
because it looks like the pattern.

```go
// WRONG — package forge (outer) declares the interface it implements.
package forge

type ChangeSource interface { /* ... */ }   // interface in the outer package
type GitHubSource struct{}                   // and its only implementation
```

```go
// Symptom: the inner package must now import `forge` to name the type, so the
// dependency still points outward. The interface belongs to the side that USES it.
```

**Passing framework or persistence types across the boundary.**

```go
// WRONG — an ORM row reaching a use case.
func (u *RunGate) Execute(ctx context.Context, row *sqlc.PullRequestRow) error
// Symptom: a schema change is now a change to the use case's signature. Pass a
// plain struct shaped for the receiving side instead.
```

**Pass-through layers.** A layer that only forwards calls, adding no translation or
isolation, is ceremony — it costs a hop and a file and buys nothing. Either it converts
something or it should not exist.

**A `shared` or `common` package everything depends on.** It acquires dependencies in every
direction and becomes a second, undirected coupling graph running under the intended one. The
global standards ban these names outright.

**Trade-off:** more code — an interface, an adapter and a translation at each boundary, none
of which does business work. That is the cost, paid for testability and for the freedom to
change infrastructure. `coupling.md` is explicit that this trade is usually right at a
boundary and wrong everywhere else.

**Related:** `ports-and-adapters.md`, `layering.md`, `coupling.md`
