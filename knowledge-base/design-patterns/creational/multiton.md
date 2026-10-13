# Multiton

*Creational · **not GoF** · Authored for this project*

## What it is

Singleton generalised from one instance to one instance **per key**. A registry maps keys to
instances and hands back the existing instance for a key or creates it on first request. Each
instance has its own state; the guarantee is uniqueness within the key, not globally
([Wikipedia](https://en.wikipedia.org/wiki/Multiton_pattern)).

No source consulted states its category outright; it is grouped as creational here because it
generalises a creational pattern.

## When to use it

- The key set is **small, known and bounded** — one instance per supported locale, per
  configured region, per registered policy family.
- Instances are expensive and genuinely must be shared per key, where two instances for one
  key would be incorrect rather than merely wasteful.
- The keyed instances are long-lived for the process's lifetime, so no eviction policy is
  needed.

## How to use it

Make the key type closed, make the map's growth bounded by that type, and construct under a
lock that cannot admit two instances for one key.

```go
type Region string                       // closed set, not an open string

var (
    mu      sync.Mutex
    clients = map[Region]*Client{}
)

func For(r Region) (*Client, error) {
    mu.Lock(); defer mu.Unlock()
    if c, ok := clients[r]; ok { return c, nil }
    c, err := newClient(r)
    if err != nil { return nil, err }    // a failure is not cached
    clients[r] = c
    return c, nil
}
```

The invariant: **the key space is bounded by something other than caller input.** The moment
callers can invent keys, this is a cache.

## How not to use it

- **With an unbounded or caller-supplied key.** `For(tenantID)` with no eviction is a memory
  leak, and every argument for a cache's size limit and expiry now applies. If you need those,
  build a cache and call it one.
- **Caching the failure.** Storing a nil or broken instance for a key that failed to
  construct makes the first transient error permanent for the process's lifetime.
- **As a service locator.** Keying by interface name or string identifier turns this into
  global dependency lookup, inheriting every objection in `singleton.md` and losing the
  compiler's help as well.
- **Assuming per-key uniqueness implies thread safety.** One instance shared across
  goroutines for a key still needs its own internal synchronisation; the registry's lock
  protects the map, not the instance.

**Trade-off:** it has all of Singleton's invisible-dependency problem, multiplied by a key —
and the key makes test isolation harder still, because clearing one key does not clear the
rest.

**Related:** `singleton.md`, `object-pool.md`
