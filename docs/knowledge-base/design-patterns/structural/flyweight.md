# Flyweight

*Structural · GoF · Authored for this project*

## What it is

State is split into **intrinsic** state, which is shared and immutable, and **extrinsic**
state, which the caller supplies per use. Many logical objects share one flyweight instance
holding the intrinsic part, so memory scales with the number of distinct intrinsic values
rather than with the number of logical objects. A factory hands out the shared instances.

## Why it exists

When a program holds very many objects whose state is mostly the same, memory grows with the
number of objects even though the distinct information grows only with the number of distinct
values. The flyweight stores the shared part once and has callers supply the varying part, so
memory follows the count of distinct values instead.

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

```go
// RIGHT — Glyph has no setters; position and style arrive per call.
type Glyph struct {
	symbol  rune
	outline Outline
}

func (g *Glyph) DrawAt(x, y int, style Style) { /* extrinsic state passed in */ }

type GlyphFactory struct {
	mu     sync.Mutex
	glyphs map[rune]*Glyph
}

func (f *GlyphFactory) Of(symbol rune) *Glyph {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, isInterned := f.glyphs[symbol]
	if !isInterned {
		g = newGlyph(symbol)
		f.glyphs[symbol] = g
	}
	return g
}
```

Where the intrinsic state is a single comparable value — an import path, a policy family
name — the standard library's `unique.Make` is the factory, and the `unique.Handle` it
returns is the flyweight.

The invariant: **a flyweight is immutable.** It is shared by definition, so a mutable field
is a data race and a correctness bug across unrelated callers.

## How not to use it

- **Applying it without measuring.** This is the most situational pattern in the catalogue,
  and it trades clear code for memory. Without a profile showing object count dominating, it
  is a complication.

```go
// WRONG — interning a few dozen check names, with no profile asking for it.
type CheckRun struct {
	Name unique.Handle[string]
}
// Symptom: every read of a name is .Value(), and the heap profile never mentioned names.
```

- **Mutable intrinsic state.** One caller's change silently alters every other user. In a
  concurrent program this is also a race, and the resulting bug is maximally hard to trace.

```go
// WRONG — a setter on a shared glyph.
func (g *Glyph) SetOutline(o Outline) { g.outline = o }
// Symptom: one caller's change redraws every other caller's text, and the race detector fires.
```

- **Letting extrinsic state creep into the shared object.** The moment position, owner or
  request context is stored on the flyweight, sharing becomes incorrect — and it will appear
  to work under single-threaded testing.

```go
// WRONG — per-use position stored on the shared glyph.
func (g *Glyph) DrawAt(x, y int, style Style) {
	g.x, g.y = x, y
	g.render(style)
}
// Symptom: concurrent draws land at each other's positions; single-threaded tests pass.
```

- **An unbounded intern cache keyed on untrusted input.** That is a memory leak reachable
  from outside. Bound the key space or bound the cache.

```go
// WRONG — every path in a webhook payload becomes a permanent factory entry.
for _, file := range payload.Files {
	refs = append(refs, paths.Of(file.Path))
}
// Symptom: a sender varying paths grows the factory's map without limit.
```

- **Confusing it with a cache or a pool.** A cache avoids recomputation; a pool reuses
  expensive mutable instances one caller at a time; a flyweight is shared *simultaneously* by
  many and therefore must be immutable.

```go
// WRONG — a pooled, mutable buffer shared by two callers at once, as if it were a flyweight.
buf := bufferPool.Get().(*bytes.Buffer)
go renderSummary(buf, decisions)
go renderAnnotations(buf, findings)
// Symptom: the two goroutines interleave writes into one buffer.
```

**Trade-off:** the object model stops matching the domain — a logical object is now a
flyweight plus parameters — and every call site carries state the object used to hold.

**Related:** `../creational/object-pool.md`, `../creational/prototype.md`, `proxy.md`
