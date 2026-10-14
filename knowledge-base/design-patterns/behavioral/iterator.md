# Iterator

*Behavioral · GoF · Authored for this project*

## What it is

Traversal is lifted out of a collection into a separate object that knows how to advance
through it and report when it is done. Clients walk the elements through the iterator's
interface without knowing whether the underlying structure is an array, a tree, a cursor over
a file, or a paginated remote API.

## When to use it

- Clients must traverse a structure **without depending on its representation**, so the
  representation can change.
- A structure needs more than one traversal order — in-order and breadth-first over the same
  tree.
- Several independent traversals must be in progress at once, each with its own position.
- The sequence is lazy, remote or infinite, and materialising it is impossible or wasteful.

## How to use it

In almost every modern language, **implement the language's own iteration protocol** rather
than inventing an interface. The pattern is built in, and using it gets the ecosystem's
`for`-loops, comprehensions and combinators for free. In Go that protocol is range-over-func:
return an `iter.Seq` or `iter.Seq2` from the `iter` package.

```go
// RIGHT — a lazy sequence; each error travels with the element it interrupted.
func (c *ForgeClient) ChangedFiles(ctx context.Context, pr PullRequest) iter.Seq2[ChangedFile, error] {
    return func(yield func(ChangedFile, error) bool) {
        for cursor := ""; ; {
            page, err := c.fetchPage(ctx, pr, cursor)
            if err != nil {
                yield(ChangedFile{}, err)
                return
            }
            for _, f := range page.Files {
                if !yield(f, nil) {
                    return
                }
            }
            if page.Next == "" {
                return
            }
            cursor = page.Next
        }
    }
}
```

The invariant: **an iterator does not own the collection's lifetime and does not mutate it.**
Its position is its only state.

## How not to use it

- **Hand-rolling `hasNext`/`next` where the language has a protocol.** You lose every
  built-in construct and gain a bespoke interface nobody else's code understands.

```go
// WRONG — a bespoke protocol where range-over-func exists.
type FileIterator interface {
    HasNext() bool
    Next() ChangedFile
}
// Symptom: no for-range, no slices.Collect, and nowhere for an error to go.
```

- **Mutating the collection during iteration.** Behaviour ranges from a thrown exception to
  skipped elements depending on the structure, and the skipped-element case is silent. Collect
  the changes and apply them after, or iterate a snapshot.

```go
// WRONG — deleting from the slice being ranged over.
for i, f := range files {
    if f.Generated {
        files = slices.Delete(files, i, i+1)
    }
}
// Symptom: the element after each deletion is skipped, the tail reads as zero values,
// and an index past the shrunken length panics. slices.DeleteFunc does this correctly.
```

- **An iterator that swallows errors.** A lazy sequence over a network or a file *will* fail
  mid-traversal. An iterator that ends quietly on error is indistinguishable from one that
  finished, and the caller processes a truncated set believing it is complete. Surface the
  failure.

```go
// WRONG — inside the sequence, a failed page ends it the way the last page does.
page, err := c.fetchPage(ctx, pr, cursor)
if err != nil {
    return
}
// Symptom: policy is evaluated over a truncated diff, and the decision looks complete.
```

- **Hiding unbounded cost behind a loop.** A `for` over a paginated API looks like a local
  loop and is a thousand round trips. If the cost is remote, the interface should say so.

```go
// WRONG — nothing in the signature says each step may be a forge round trip.
func (pr PullRequest) Files() iter.Seq[ChangedFile]
// Symptom: a loop that reads as local makes a thousand requests, with no ctx to cancel them.
```

- **Returning an iterator from a method that holds a lock.** The lock's scope now ends
  wherever the caller happens to stop iterating.

```go
// WRONG — the read lock is held for as long as the caller's loop body runs.
func (r *Registry) Policies() iter.Seq[Policy] {
    return func(yield func(Policy) bool) {
        r.mu.RLock()
        defer r.mu.RUnlock()
        for _, p := range r.policies {
            if !yield(p) {
                return
            }
        }
    }
}
// Symptom: a slow loop body stalls every writer; one that writes to the Registry deadlocks.
```

**Trade-off:** laziness moves failure and cost away from the call that created the iterator
and into the loop that consumes it, which makes both harder to attribute.

**Related:** `../structural/composite.md`, `visitor.md`
