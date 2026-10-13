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
`for`-loops, comprehensions and combinators for free.

```python
class PaginatedFiles:
    def __iter__(self):
        cursor = None
        while True:
            page = self._fetch(cursor)
            yield from page.items          # laziness is the point
            if not page.next: return
            cursor = page.next
```

The invariant: **an iterator does not own the collection's lifetime and does not mutate it.**
Its position is its only state.

## How not to use it

- **Hand-rolling `hasNext`/`next` where the language has a protocol.** You lose every
  built-in construct and gain a bespoke interface nobody else's code understands.
- **Mutating the collection during iteration.** Behaviour ranges from a thrown exception to
  skipped elements depending on the structure, and the skipped-element case is silent. Collect
  the changes and apply them after, or iterate a snapshot.
- **An iterator that swallows errors.** A lazy sequence over a network or a file *will* fail
  mid-traversal. An iterator that ends quietly on error is indistinguishable from one that
  finished, and the caller processes a truncated set believing it is complete. Surface the
  failure.
- **Hiding unbounded cost behind a loop.** A `for` over a paginated API looks like a local
  loop and is a thousand round trips. If the cost is remote, the interface should say so.
- **Returning an iterator from a method that holds a lock.** The lock's scope now ends
  wherever the caller happens to stop iterating.

**Trade-off:** laziness moves failure and cost away from the call that created the iterator
and into the loop that consumes it, which makes both harder to attribute.

**Related:** `../structural/composite.md`, `visitor.md`
