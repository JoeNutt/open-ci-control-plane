# Bridge

*Structural · GoF · Authored for this project*

## What it is

An abstraction and its implementation are put into separate hierarchies and connected by
composition, so each can be extended without touching the other. The abstraction holds a
reference to an implementor interface and delegates the primitive operations to it, while
adding its own higher-level behaviour on top.

## When to use it

- **Two independent dimensions of variation** are multiplying subclasses. Three report kinds
  across four output devices is twelve classes by inheritance and seven by bridge.
- The implementation must be selectable or replaceable at runtime, not fixed at compile time.
- An abstraction's clients should be unaffected when an implementation is added — a new
  storage backend, a new rendering target.

## How to use it

Keep the implementor interface **primitive**. It exposes the small operations the abstraction
composes; it does not mirror the abstraction's API.

```kotlin
interface Renderer {                 // implementor: primitives only
    fun drawLine(x1: Int, y1: Int, x2: Int, y2: Int)
    fun fill(colour: Colour)
}

abstract class Shape(protected val renderer: Renderer) {   // abstraction
    abstract fun draw()              // composed from primitives
}

class Rectangle(renderer: Renderer) : Shape(renderer) {
    override fun draw() { /* four drawLine calls, then fill */ }
}
```

The invariant: **neither hierarchy names a concrete type from the other.** Adding a renderer
must require no change to any shape, and adding a shape no change to any renderer.

## How not to use it

- **Applying it with one implementation.** With a single implementor the indirection buys
  nothing and costs a layer. Introduce it when the second dimension actually appears, not in
  anticipation.
- **An implementor interface that mirrors the abstraction method for method.** Then it is not
  a bridge, it is a delegation layer, and both hierarchies change together — which is the
  exact coupling the pattern exists to break.
- **Confusing it with Adapter.** Bridge is designed up front to let two hierarchies vary;
  Adapter is applied afterwards to reconcile interfaces that already exist and disagree.
- **Confusing it with Strategy.** Structurally they are near-identical; the intent differs.
  Strategy swaps an algorithm behind one abstraction; Bridge exists so two hierarchies can
  grow independently. If there is only one abstraction, you have a Strategy.
- **Letting state leak across the bridge.** If the abstraction reaches into implementor state,
  or the implementor calls back into the abstraction's higher-level methods, the separation is
  nominal.

**Trade-off:** one more indirection on every primitive call, and a design that is harder to
follow than inheritance until the second dimension justifies it.

**Related:** `adapter.md`, `../behavioral/strategy.md`
