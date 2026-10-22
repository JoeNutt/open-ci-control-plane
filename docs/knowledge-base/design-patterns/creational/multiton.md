# Multiton

*Creational · **not GoF** · Authored for this project*

## What it is

Singleton generalised from one instance to one instance **per key**. A registry maps keys to
instances and hands back the existing instance for a key or creates it on first request. Each
instance has its own state; the guarantee is uniqueness within the key, not globally
([Wikipedia](https://en.wikipedia.org/wiki/Multiton_pattern)).

No source consulted states its category outright; it is grouped as creational here because it
generalises a creational pattern.

## Why it exists

Some resources must exist exactly once per key — one client per region, one compiled policy
per family — because two instances for the same key would hold divergent state or contend
for what the key represents. Constructing them wherever they are needed produces duplicates,
and a single shared instance cannot represent several keys. The registry resolves both:
construct on first request for a key, return that instance ever after.

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
// RIGHT — a closed key type, construction under the lock, failures not cached.
type Region string

var (
	mu      sync.Mutex
	clients = map[Region]*Client{}
)

func For(r Region) (*Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := clients[r]; ok {
		return c, nil
	}
	c, err := newClient(r)
	if err != nil {
		return nil, err
	}
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

```go
// WRONG — the key is caller input, and nothing is ever removed.
func ForTenant(tenantID string) *Client {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := tenantClients[tenantID]; !ok {
		tenantClients[tenantID] = newTenantClient(tenantID)
	}
	return tenantClients[tenantID]
}
// Symptom: the heap grows with every tenant ID ever sent, including ones a caller invented.
```

- **Caching the failure.** Storing a nil or broken instance for a key that failed to
  construct makes the first transient error permanent for the process's lifetime.

```go
// WRONG — the result of a failed construction is stored.
c, err := newClient(r)
clients[r] = c
return c, err
// Symptom: after one transient failure, every later For(r) returns a nil client and no error.
```

- **As a service locator.** Keying by interface name or string identifier turns this into
  global dependency lookup, inheriting every objection in `singleton.md` and losing the
  compiler's help as well.

```go
// WRONG — keyed by name, returning anything.
func Lookup(name string) any { return services[name] }
// Symptom: Lookup("policyStore").(PolicyStore) compiles, and a misspelt name panics at runtime.
```

- **Assuming per-key uniqueness implies thread safety.** One instance shared across
  goroutines for a key still needs its own internal synchronisation; the registry's lock
  protects the map, not the instance.

```go
// WRONG — the registry's lock is assumed to cover the instance.
func (c *Client) Remember(etag string) { c.etags = append(c.etags, etag) }
// Symptom: two goroutines calling Remember on one region's client race; go test -race reports it.
```

**Trade-off:** it has all of Singleton's invisible-dependency problem, multiplied by a key —
and the key makes test isolation harder still, because clearing one key does not clear the
rest.

**Related:** `singleton.md`, `object-pool.md`
