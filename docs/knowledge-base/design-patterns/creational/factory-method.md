# Factory Method

*Creational · GoF · Authored for this project*

## What it is

A class declares a method that returns a product, and defers the choice of concrete product
to its subclasses. The base class writes its algorithm against the product's interface and
calls its own factory method wherever it needs an instance; each subclass overrides that
method to return the type it wants. The decision moves from the point of use to the point of
subclassing.

## Why it exists

An algorithm that must create a collaborator, and calls a concrete constructor to do it, fixes
that type for every variant of the algorithm. Varying it then means copying the algorithm, or
adding a conditional over types at the creation site that grows with every new product. The
pattern has the algorithm ask for the product through a method, so the choice of concrete type
moves to whoever supplies that method and the algorithm is written once.

## When to use it

- A base class contains logic that must create collaborators whose concrete types it cannot
  know, and subclasses legitimately differ in that choice.
- A framework defines a workflow and lets callers supply the participants without rewriting
  the workflow.
- The set of products grows by adding subclasses rather than by editing a conditional.

## How to use it

Keep the factory method **protected and abstract**, return the interface, and let the base
class's algorithm be the only caller.

Go has no subclassing and no `protected`, so the classical shape does not exist, and
embedding does not supply it: a method on an embedded struct that calls another method
dispatches to the embedded type, never to the outer one. The Go form holds the factory as an
unexported func field, supplied once at construction, with the algorithm as its only caller.

```go
// RIGHT — Run never names a concrete Formatter; the constructor chose it.
type ReportJob struct {
	newFormatter func() Formatter
}

func NewCSVReportJob() ReportJob {
	return ReportJob{newFormatter: func() Formatter { return csvFormatter{} }}
}

func (j ReportJob) Run(d Decision) string {
	return j.newFormatter().Format(d)
}
```

The invariant: **the base class must not contain a branch over product types.** If it does,
the decision has not actually moved and the pattern is decoration.

## How not to use it

- **A factory method with a parameter that selects the type.** `createFormatter(String kind)`
  with a switch inside is a Simple Factory wearing this pattern's name. It centralises the
  conditional rather than removing it, and every new product edits that method.

```go
// WRONG — the parameter is the conditional the pattern was meant to remove.
func newFormatter(kind string) Formatter {
	switch kind {
	case "csv":
		return csvFormatter{}
	case "json":
		return jsonFormatter{}
	}
	return nil
}
// Symptom: every new format edits this switch, and an unknown kind is a nil far from its cause.
```

- **Subclassing purely to vary one object.** If the subclass overrides nothing but the
  factory method, pass the product in instead — a constructor parameter or an injected
  factory function is the same decision with no inheritance.

```go
// WRONG — a type per product, overriding nothing but the factory.
type CSVReportJob struct{ ReportJob }

func (CSVReportJob) newFormatter() Formatter { return csvFormatter{} }
// Symptom: one type per format, and ReportJob.Run never calls this method.
```

- **In a language with first-class functions, reaching for a class hierarchy at all.** A
  `func() Formatter` field is the whole pattern. The hierarchy earns its place only when the
  base class has real algorithm to protect.

```go
// WRONG — an interface and a type per product, standing in for a func value.
type FormatterFactory interface{ NewFormatter() Formatter }

type csvFormatterFactory struct{}

func (csvFormatterFactory) NewFormatter() Formatter { return csvFormatter{} }
// Symptom: two declarations per format, protecting no algorithm.
```

- **Confusing it with Abstract Factory.** This creates *one* product; Abstract Factory
  creates a *family* whose members must match.

```go
// WRONG — two factory methods whose products must agree, and nothing makes them.
type ReportJob struct {
	newFormatter func() Formatter
	newParser    func() Parser
}
// Symptom: a CSV formatter paired with a JSON parser compiles and fails at run time.
```

**Trade-off:** one class per product variation, and the product choice becomes visible in
the type hierarchy rather than at the call site — which is harder to trace.

**Related:** `abstract-factory.md`, `prototype.md`, `../behavioral/template-method.md`
