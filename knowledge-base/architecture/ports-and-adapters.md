# Ports and adapters

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

A **port** is an interface owned by the inside — the domain declares what it needs or what it
offers. An **adapter** lives outside and implements that port for one specific technology.
**Driving** adapters call in (HTTP, CLI, tests, message consumers); **driven** adapters are
called by the domain (databases, external services, clocks).

It is the same rule as `dependency-direction.md` in more usable vocabulary. Clean Architecture
is a synthesis of this (Cockburn, 2005) and the onion (Palermo, 2008) — not a competing
choice.

## Why it exists

Counting rings does not answer "where does this go?". Asking *is this a port, an adapter, or
domain?* does, and it makes the crucial point explicit in the vocabulary itself: **the
interface belongs to the inner side, not to the implementer.**

It also names the thing that makes a domain testable. Every non-deterministic dependency —
clock, randomness, identifiers, filesystem, network — behind a port is a domain that needs no
infrastructure to test.

## When to use it

- At the boundaries **between** services or modules, as a translation layer.
- Where a dependency is volatile and the domain must not know about it.
- For anything non-deterministic the domain touches.
- The strongest version aligns boundaries with **bounded contexts**: areas where the
  vocabulary genuinely differs, translating when information crosses.

**Not uniformly at every seam inside one module.** Applied everywhere it is ceremony.

## How to use it

Name the port for what the domain needs, keep it to the subset actually used, and let the
adapter absorb every difference — naming, units, error shape, pagination.

```go
// RIGHT — package policy. Named for the need, not the implementer.
// Two methods, because two are used. Not a mirror of any SDK.
type ChangeSource interface {
    FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error)
    HeadSHA(ctx context.Context, pr PullRequest) (string, error)
}

// Non-determinism behind a port too, so the domain is testable with no infrastructure.
type Clock interface{ Now() time.Time }
```

```go
// RIGHT — the test needs no mock framework and no network.
type fakeSource struct{ files []policy.ChangedFile }

func (f fakeSource) FilesChanged(context.Context, policy.PullRequest) ([]policy.ChangedFile, error) {
    return f.files, nil
}
```

**The invariant: no vendor type crosses the port, in either direction.** A port is nearly
always a *simpler* thing than the API behind it, expressed in your vocabulary. Done properly,
the inside does not know the vendor exists.

## How not to use it

**A port that mirrors the SDK.**

```go
// WRONG — the port was designed by reading the vendor's API.
type GitHubGateway interface {
    ListPullRequestFiles(ctx context.Context, owner, repo string, number int,
        opts *github.ListOptions) ([]*github.CommitFile, *github.Response, error)
    GetCheckRun(ctx context.Context, owner, repo string, id int64) (*github.CheckRun, error)
}
// Symptom: named for the implementer, carries vendor types and vendor pagination.
// Swapping forge rewrites the domain. This renames the coupling; it does not remove it.
```

**A single wide interface instead of one port per concern.**

```go
// WRONG — forty methods behind one name is not a boundary.
type Database interface {
    GetPolicy(...) ; SavePolicy(...) ; GetDecision(...) ; SaveDecision(...)
    GetGrant(...)  ; SaveGrant(...)  ; /* ...34 more... */
}
// Symptom: every consumer depends on everything, and a fake for one test must
// implement all forty. Split by the concern the caller actually has.
```

**The adapter carrying business rules.**

```go
// WRONG — the adapter decided something.
func (s GitHubSource) FilesChanged(ctx context.Context, pr policy.PullRequest) ([]policy.ChangedFile, error) {
    files := s.fetch(ctx, pr)
    if len(files) > 1000 {
        return nil, errors.New("too many files")   // a policy decision, outside the domain
    }
    return files, nil
}
// Symptom: the rule is unreachable from a domain test and invisible to the gate.
// Deciding HOW to ask the forge belongs here; deciding WHAT IT MEANS does not.
```

**Translating the happy path only.** Errors, timeouts, pagination, rate limits and partial
results are part of the interface. An adapter forwarding a raw vendor error has moved the
coupling into the error-handling code, where it is harder to see.

**Applying it at every internal seam.** A port between two packages that are always changed
together is one module with extra ceremony — see `module-boundaries.md`.

**Trade-off:** a mapping layer per boundary that must be kept honest as the outside drifts,
and an extra set of types that do no business work.

**Related:** `dependency-direction.md`, `layering.md`,
`../design-patterns/structural/adapter.md`
