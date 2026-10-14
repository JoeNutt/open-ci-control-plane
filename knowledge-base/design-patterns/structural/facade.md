# Facade

*Structural · GoF · Authored for this project*

## What it is

A single object provides a simplified interface to a set of interfaces in a subsystem. It
knows which subsystem objects to call, in what order, and with what defaults, and exposes
one operation that means something to the caller. The subsystem remains fully accessible to
anyone who needs more.

## When to use it

- A common path through a complicated subsystem is reproduced by every caller, slightly
  differently, which is duplicated knowledge of the subsystem's sequencing.
- A subsystem's interface is correct but low-level, and most callers want one named outcome
  rather than nine primitives.
- A layer boundary needs a deliberately narrow entry point so what callers depend on is
  visible and small.

## How to use it

Name the facade for the **outcome**, not the subsystem, and let it hold the sequencing
knowledge and nothing else.

```go
// RIGHT — named for what it produces; it holds the order and nothing else.
type ChangeIngest struct {
    forge      ChangeSource
    extractors ImportExtractors
    bounds     Bounds
}

func (c ChangeIngest) BuildDocument(ctx context.Context, e ForgeEvent) (ChangeDocument, error) {
    files, err := c.forge.FilesChanged(ctx, e.PullRequest)
    if err != nil {
        return ChangeDocument{}, err
    }
    if err := c.bounds.Check(files); err != nil { // order matters, and lives here
        return ChangeDocument{}, err
    }
    imports, err := c.extractors.Extract(files)
    if err != nil {
        return ChangeDocument{}, err
    }
    return assemble(e, files, imports), nil
}
```

The invariant: **the facade adds no rules of its own.** It sequences and defaults. The moment
it decides something the subsystem does not, it is a new component and should be named as one.

## How not to use it

- **Letting it accumulate until it is the subsystem.** The usual failure: every new caller
  adds a method, and the facade becomes a god object that knows everything and is changed by
  every feature. Watch for methods whose field usage does not overlap — that is two facades.

```go
// WRONG — every new caller added a method.
func (c ChangeIngest) BuildDocument(ctx context.Context, e ForgeEvent) (ChangeDocument, error)
func (c ChangeIngest) RedeliverWebhook(ctx context.Context, id DeliveryID) error
func (c ChangeIngest) RotateForgeToken(ctx context.Context) error
// Symptom: changed by every feature, and each method uses a different subset of the fields.
```

- **Hiding the subsystem rather than simplifying it.** If the facade is the only legal way in,
  it must cover every case, and it will grow until it does. Leave the subsystem reachable.

```go
// WRONG — the forge client is unexported, so the facade grows a variant per need.
func (c ChangeIngest) BuildDocument(ctx context.Context, e ForgeEvent) (ChangeDocument, error)
func (c ChangeIngest) BuildDocumentForPaths(ctx context.Context, e ForgeEvent, paths []string) (ChangeDocument, error)
func (c ChangeIngest) FilesOnly(ctx context.Context, e ForgeEvent) ([]ChangedFile, error)
// Symptom: a caller needing one primitive waits for a new facade method.
```

- **Pass-through methods.** A facade method that forwards one call unchanged adds a name and
  a hop. Either it sequences something or it should not exist.

```go
// WRONG — forwards one call unchanged.
func (c ChangeIngest) FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error) {
    return c.forge.FilesChanged(ctx, pr)
}
// Symptom: a name and a hop that sequence nothing.
```

- **Putting a facade over a subsystem you own and could simplify directly.** The pattern
  earns its place against complexity you cannot change; against your own code it is often
  cheaper to fix the interface.

```go
// WRONG — a facade smoothing over an awkward interface this team owns.
func (c ChangeIngest) Extract(files []ChangedFile) ([]Ref, error) {
    c.extractors.Init()
    defer c.extractors.Reset()
    return c.extractors.ExtractWithOptions(files, DefaultExtractOptions())
}
// Symptom: every other caller still meets Init and Reset; fixing the extractors was cheaper.
```

- **Confusing it with Adapter.** Adapter satisfies an interface the client already requires;
  Facade invents a more convenient one.

```go
// WRONG — named a facade, but it only satisfies a port the domain already declared.
type forgeFacade struct{ api *github.Client }

func (f forgeFacade) FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error)
// Symptom: readers look for sequencing it does not have; the name hides an Adapter.
```

**Trade-off:** one more place to change when the subsystem changes, and a tendency to grow
that must be actively resisted.

**Related:** `adapter.md`, `proxy.md`
