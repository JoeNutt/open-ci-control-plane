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

## Why it exists

Some objects cost far more to create than to use — a TLS session, a database connection — and
creating one per unit of work spends most of the time on setup. Creating them without limit
also exhausts what the other end will accept. The pool pays creation once per object and
bounds how many exist, at the price of owning each object's clean state between users.

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
// RIGHT — bounded, acquisition honours a deadline, reset on return.
type Pool struct{ free chan *Conn }

func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
	select {
	case c := <-p.free:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) Release(c *Conn) {
	c.Reset()
	select {
	case p.free <- c:
	default:
		c.Close()
	}
}
```

Waiting is bounded by the caller's context, never an unbounded queue; a release beyond
capacity closes the object rather than growing the pool.

The invariant: **an object taken from the pool is indistinguishable from a fresh one.** Any
state surviving from the previous user is a leak between unrelated pieces of work.

## How not to use it

- **Pooling cheap objects.** Pooling plain structs or short-lived values usually loses to the
  allocator and defeats generational garbage collection, while adding a synchronisation point
  every allocation must pass through. Measure first; this is the most common misuse.

```go
// WRONG — pooling a value cheaper to allocate than to synchronise on.
func acquireLabel() *Label {
	select {
	case l := <-labels:
		return l
	default:
		return &Label{}
	}
}
// Symptom: a benchmark shows acquireLabel slower than &Label{}, and every caller contends on one channel.
```

- **Resetting on acquire instead of on return.** A dirty object sits in the pool between
  uses, so any code that inspects the pool, or any path that skips the reset, leaks the
  previous user's data. On a shared service that is a cross-request information leak.

```go
// WRONG — released dirty; cleaned only when the next caller takes it.
func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
	select {
	case c := <-p.free:
		c.Reset()
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
// Symptom: a dump of the idle pool shows the previous request's credentials.
```

- **An unbounded pool.** That is a memory leak with a queue in front of it. The bound is the
  feature.

```go
// WRONG — Release keeps everything it is given.
func (p *Pool) Release(c *Conn) {
	c.Reset()
	p.idle = append(p.idle, c)
}
// Symptom: after one traffic spike the pool holds every connection ever opened, and none closes.
```

- **Pooling objects that carry identity or per-caller state** — anything with a user, tenant
  or transaction attached. Interchangeability is a precondition, not an aspiration.

```go
// WRONG — the pooled object remembers who acquired it.
type Session struct {
	conn   *Conn
	tenant TenantID
}
// Symptom: a session released by one tenant and acquired by another still acts as the first.
```

- **Forgetting the path that never returns the object.** An early return, a panic or a
  cancelled context that skips `Release` exhausts the pool, and the symptom is a hang rather
  than an error. Release with `defer` immediately after a successful acquire.

```go
// WRONG — the early return skips Release.
c, err := pool.Acquire(ctx)
if err != nil {
	return err
}
if err := c.Send(msg); err != nil {
	return err
}
pool.Release(c)
// Symptom: each failed Send leaks a connection; once all have leaked, Acquire waits out every deadline.
```

**Trade-off:** the pool becomes a shared, synchronised, stateful component on the hot path —
and a bug in it is a bug in everything that uses it.

**Related:** `singleton.md`, `../structural/flyweight.md`
