# Adapter

*Structural · GoF · Authored for this project*

## What it is

A wrapper that translates between the interface a client requires and the interface an
existing type offers. The adapter implements the target interface, holds the adaptee, and
converts calls, arguments and errors across the boundary. Neither side changes; the
incompatibility is absorbed in one place.

## When to use it

- A third-party or legacy type does the right work with the wrong shape, and you cannot or
  should not change it.
- Your own code must depend on an interface it owns, while the implementation lives outside —
  the **ports and adapters** arrangement, where the port is the interface the domain declares
  and the adapter is the translation to the outside world.
- Several interchangeable providers must present one interface to the domain — two forges,
  three object stores, four import extractors.

## How to use it

Let the **consuming** side own the interface, and put every translation — naming, units,
error shape, null handling — inside the adapter so none of it leaks inward.

```go
// The domain declares what it needs. It names no vendor type.
type ChangeSource interface {
    FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error)
}

// The adapter translates the forge's API into that vocabulary.
type gitHubChangeSource struct{ api *github.Client }

func (g gitHubChangeSource) FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error) {
    raw, _, err := g.api.PullRequests.ListFiles(ctx, pr.Owner, pr.Repo, pr.Number, nil)
    if err != nil {
        return nil, fmt.Errorf("list files: %w", translateForgeError(err))  // error shape too
    }
    return toChangedFiles(raw), nil
}
```

The invariant: **no vendor type crosses the adapter.** If a `github.*` type appears in a
domain signature, the adapter has failed and the dependency has leaked.

## How not to use it

- **Letting the adaptee's interface define the port.** An interface shaped like the vendor's
  API, with its field names and error codes, means swapping providers rewrites the domain.
  The port is designed from what the domain needs, before looking at any SDK.
- **Translating the happy path only.** Errors, timeouts, pagination, rate limits and partial
  results are all part of the interface. An adapter that forwards a raw vendor error has
  moved the coupling rather than removed it.
- **Putting business logic in the adapter.** Deciding *what to do* with a change belongs
  inside; deciding *how to ask the forge* belongs here. An adapter with a policy decision in
  it is two components.
- **One adapter per method.** A class whose only job is to rename a single call is noise;
  call the thing directly or widen the adapter's responsibility to the whole boundary.
- **Confusing it with Facade.** Adapter converts an interface the client already requires;
  Facade invents a simpler one the client would prefer.

**Trade-off:** an extra layer, an extra set of types, and a mapping that must be kept honest
as the outside interface drifts.

**Related:** `facade.md`, `proxy.md`, `bridge.md`
