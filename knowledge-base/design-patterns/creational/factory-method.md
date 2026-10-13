# Factory Method

*Creational · GoF · Authored for this project*

## What it is

A class declares a method that returns a product, and defers the choice of concrete product
to its subclasses. The base class writes its algorithm against the product's interface and
calls its own factory method wherever it needs an instance; each subclass overrides that
method to return the type it wants. The decision moves from the point of use to the point of
subclassing.

## When to use it

- A base class contains logic that must create collaborators whose concrete types it cannot
  know, and subclasses legitimately differ in that choice.
- A framework defines a workflow and lets callers supply the participants without rewriting
  the workflow.
- The set of products grows by adding subclasses rather than by editing a conditional.

## How to use it

Keep the factory method **protected and abstract**, return the interface, and let the base
class's algorithm be the only caller.

```java
abstract class ReportJob {
    protected abstract Formatter createFormatter();   // the factory method

    final String run(Data data) {
        return createFormatter().format(data);        // base class never names a concrete type
    }
}

final class CsvReportJob extends ReportJob {
    protected Formatter createFormatter() { return new CsvFormatter(); }
}
```

The invariant: **the base class must not contain a branch over product types.** If it does,
the decision has not actually moved and the pattern is decoration.

## How not to use it

- **A factory method with a parameter that selects the type.** `createFormatter(String kind)`
  with a switch inside is a Simple Factory wearing this pattern's name. It centralises the
  conditional rather than removing it, and every new product edits that method.
- **Subclassing purely to vary one object.** If the subclass overrides nothing but the
  factory method, pass the product in instead — a constructor parameter or an injected
  factory function is the same decision with no inheritance.
- **In a language with first-class functions, reaching for a class hierarchy at all.** A
  `func() Formatter` field is the whole pattern. The hierarchy earns its place only when the
  base class has real algorithm to protect.
- **Confusing it with Abstract Factory.** This creates *one* product; Abstract Factory
  creates a *family* whose members must match.

**Trade-off:** one class per product variation, and the product choice becomes visible in
the type hierarchy rather than at the call site — which is harder to trace.

**Related:** `abstract-factory.md`, `prototype.md`, `../behavioral/template-method.md`
