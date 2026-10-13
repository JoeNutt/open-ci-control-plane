# Flyweight

*Structural · GoF · Authored for this project*

## What it is

State is split into **intrinsic** state, which is shared and immutable, and **extrinsic**
state, which the caller supplies per use. Many logical objects share one flyweight instance
holding the intrinsic part, so memory scales with the number of distinct intrinsic values
rather than with the number of logical objects. A factory hands out the shared instances.

## When to use it

- Object **count** is the problem, not object cost — hundreds of thousands of them, and the
  heap profile proves it.
- A large majority of each object's state is duplicated across instances: a glyph's outline,
  a tile's texture, a token's kind, an import path repeated across thousands of files.
- The duplicated state is genuinely immutable, and the varying state can be passed in or held
  by the caller.

## How to use it

Make the shared object immutable, intern it through a factory, and pass extrinsic state as
parameters.

```java
final class Glyph {                      // intrinsic: shared, immutable
    private final char symbol;
    private final Outline outline;
    void drawAt(int x, int y, Style style) { /* extrinsic passed in */ }
}

final class Glyphs {
    private static final Map<Character, Glyph> cache = new ConcurrentHashMap<>();
    static Glyph of(char c) { return cache.computeIfAbsent(c, Glyph::new); }
}
```

The invariant: **a flyweight is immutable.** It is shared by definition, so a mutable field
is a data race and a correctness bug across unrelated callers.

## How not to use it

- **Applying it without measuring.** This is the most situational pattern in the catalogue,
  and it trades clear code for memory. Without a profile showing object count dominating, it
  is a complication.
- **Mutable intrinsic state.** One caller's change silently alters every other user. In a
  concurrent program this is also a race, and the resulting bug is maximally hard to trace.
- **Letting extrinsic state creep into the shared object.** The moment position, owner or
  request context is stored on the flyweight, sharing becomes incorrect — and it will appear
  to work under single-threaded testing.
- **An unbounded intern cache keyed on untrusted input.** That is a memory leak reachable
  from outside. Bound the key space or bound the cache.
- **Confusing it with a cache or a pool.** A cache avoids recomputation; a pool reuses
  expensive mutable instances one caller at a time; a flyweight is shared *simultaneously* by
  many and therefore must be immutable.

**Trade-off:** the object model stops matching the domain — a logical object is now a
flyweight plus parameters — and every call site carries state the object used to hold.

**Related:** `../creational/object-pool.md`, `../creational/prototype.md`, `proxy.md`
