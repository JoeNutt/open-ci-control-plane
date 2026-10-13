# Object Pool

*Creational · **not GoF** · Authored for this project*

## What it is

A fixed or bounded set of initialised objects is held ready for use. Callers acquire an
instance from the pool, use it, and return it rather than destroying it. Creation cost is
paid once per pooled object instead of once per use, and the pool — not the caller — owns the
lifetime.

Consistently classified as creational across catalogues
([Wikipedia](https://en.wikipedia.org/wiki/Object_pool_pattern)), though it is not one of the
23 GoF patterns.

## When to use it

- Creation is genuinely expensive **and measured to be** — a TCP or TLS handshake, a database
  session, a compiled regular expression or policy, a large preallocated buffer.
- Instances are **interchangeable**: a caller cannot tell which one it received and must not
  care.
- The expensive resource is scarce and the bound is part of the point, as with a connection
  limit the server enforces.

## How to use it

Bound the pool, make acquisition able to fail or block with a deadline, and reset state on
return rather than on acquire.

```go
type Pool struct{ free chan *Conn }

func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
    select {
    case c := <-p.free:
        return c, nil
    case <-ctx.Done():
        return nil, ctx.Err()          // bounded wait, never an unbounded queue
    }
}

func (p *Pool) Release(c *Conn) {
    c.Reset()                           // reset on return: the pool owns clean state
    select { case p.free <- c: default: c.Close() }   // over capacity, discard
}
```

The invariant: **an object taken from the pool is indistinguishable from a fresh one.** Any
state surviving from the previous user is a leak between unrelated pieces of work.

## How not to use it

- **Pooling cheap objects.** Pooling plain structs or short-lived values usually loses to the
  allocator and defeats generational garbage collection, while adding a synchronisation point
  every allocation must pass through. Measure first; this is the most common misuse.
- **Resetting on acquire instead of on return.** A dirty object sits in the pool between
  uses, so any code that inspects the pool, or any path that skips the reset, leaks the
  previous user's data. On a shared service that is a cross-request information leak.
- **An unbounded pool.** That is a memory leak with a queue in front of it. The bound is the
  feature.
- **Pooling objects that carry identity or per-caller state** — anything with a user, tenant
  or transaction attached. Interchangeability is a precondition, not an aspiration.
- **Forgetting the path that never returns the object.** An early return, a panic or a
  cancelled context that skips `Release` exhausts the pool, and the symptom is a hang rather
  than an error.

**Trade-off:** the pool becomes a shared, synchronised, stateful component on the hot path —
and a bug in it is a bug in everything that uses it.

**Related:** `singleton.md`, `../structural/flyweight.md`
