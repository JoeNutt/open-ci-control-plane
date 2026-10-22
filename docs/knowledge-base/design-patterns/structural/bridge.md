# Bridge

*Structural · GoF · Authored for this project*

## What it is

An abstraction and its implementation are put into separate hierarchies and connected by
composition, so each can be extended without touching the other. The abstraction holds a
reference to an implementor interface and delegates the primitive operations to it, while
adding its own higher-level behaviour on top.

## Why it exists

When a type varies along two independent dimensions and both are expressed as separate types,
their number is the product of the two — every new report kind needs a version per output
device, and every new device a version per report kind. The bridge splits the dimensions into
two families joined by composition, so each grows by addition and the count is their sum.

## When to use it

- **Two independent dimensions of variation** are multiplying subclasses. Three report kinds
  across four output devices is twelve classes by inheritance and seven by bridge.
- The implementation must be selectable or replaceable at runtime, not fixed at compile time.
- An abstraction's clients should be unaffected when an implementation is added — a new
  storage backend, a new rendering target.

## How to use it

Keep the implementor interface **primitive**. It exposes the small operations the abstraction
composes; it does not mirror the abstraction's API.

Go has no abstract classes, so the abstraction's "hierarchy" is a set of types satisfying one
interface — here `Report{ Render() }` — each holding the implementor as a field.

```go
// RIGHT — Sink offers primitives; each report kind composes them through the Sink it holds.
type Sink interface {
	Heading(text string)
	Row(cells ...string)
}

type DecisionReport struct {
	sink      Sink
	decisions []Decision
}

func (r DecisionReport) Render() {
	r.sink.Heading(decisionsHeading)
	for _, d := range r.decisions {
		r.sink.Row(d.Policy, d.Outcome.String())
	}
}
```

The invariant: **neither hierarchy names a concrete type from the other.** Adding a sink must
require no change to any report, and adding a report no change to any sink.

## How not to use it

- **Applying it with one implementation.** With a single implementor the indirection buys
  nothing and costs a layer. Introduce it when the second dimension actually appears, not in
  anticipation.

```go
// WRONG — the implementor interface has one implementation and no second in sight.
type Sink interface{ Heading(text string) }

type markdownSink struct{ out *strings.Builder }
// Symptom: every report call crosses an interface that has only ever had one implementation.
```

- **An implementor interface that mirrors the abstraction method for method.** Then it is not
  a bridge, it is a delegation layer, and both hierarchies change together — which is the
  exact coupling the pattern exists to break.

```go
// WRONG — the implementor mirrors the report kinds rather than offering primitives.
type Sink interface {
	RenderDecisionReport(decisions []Decision)
	RenderCoverageReport(c Coverage)
}
// Symptom: a new report kind edits every sink, and a new sink implements every report.
```

- **Confusing it with Adapter.** Bridge is designed up front to let two hierarchies vary;
  Adapter is applied afterwards to reconcile interfaces that already exist and disagree.

```go
// WRONG — named a bridge, but written afterwards to fit an existing client to Sink.
type commentBridge struct{ api *github.Client }

func (b commentBridge) Heading(text string) { /* translate to the forge's comment call */ }
// Symptom: readers look for a second hierarchy that does not exist; the name hides an Adapter.
```

- **Confusing it with Strategy.** Structurally they are near-identical; the intent differs.
  Strategy swaps an algorithm behind one abstraction; Bridge exists so two hierarchies can
  grow independently. If there is only one abstraction, you have a Strategy.

```go
// WRONG — named a bridge, though DecisionReport is the only type that will hold a Sink.
type DecisionReport struct {
	sinkBridge Sink
}
// Symptom: no second abstraction exists to vary; this is a Strategy under a heavier name.
```

- **Letting state leak across the bridge.** If the abstraction reaches into implementor state,
  or the implementor calls back into the abstraction's higher-level methods, the separation is
  nominal.

```go
// WRONG — the report recovers the concrete sink and writes into its state.
func (r DecisionReport) Render() {
	if md, isMarkdown := r.sink.(*markdownSink); isMarkdown {
		md.out.WriteString("<details>")
	}
	r.sink.Heading(decisionsHeading)
}
// Symptom: adding a sink means editing reports, and the two hierarchies change together.
```

**Trade-off:** one more indirection on every primitive call, and a design that is harder to
follow than inheritance until the second dimension justifies it.

**Related:** `adapter.md`, `../behavioral/strategy.md`
