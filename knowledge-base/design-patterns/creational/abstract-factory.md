# Abstract Factory

*Creational · GoF · Authored for this project*

## What it is

An interface declares a creation method for each member of a family of related products.
Each concrete factory implements that interface and returns products from one coherent
family. Client code holds the factory interface and the product interfaces, so it can build
a complete set of collaborating objects without naming any concrete class and without being
able to mix families by accident.

## Why it exists

When a system must work with one of several families of related objects, constructing each
product directly scatters the choice of family across every creation site. Each site then
chooses on its own, and nothing stops one creating an S3 reader while another creates a
local-disk locker — a combination that compiles and fails at runtime, or fails silently. The
pattern moves the choice of family into one object, so it is made once and every product
follows it.

## When to use it

- Products are used **together** and a mismatched combination is a defect — a Windows
  scrollbar inside a GTK window, a Postgres statement builder paired with a MySQL dialect.
- The family is swapped wholesale: per deployment target, per tenant, per supported backend.
- The set of *products in the family* is stable, and the set of *families* is what grows.

## How to use it

Give the factory one method per product, and let the family boundary be the thing that
cannot be crossed.

```go
// RIGHT — one method per product; the concrete factory is the family.
type StorageFactory interface {
	NewReader() Reader
	NewWriter() Writer
	NewLocker() Locker
}

type s3Factory struct{ bucket string }

func (f s3Factory) NewReader() Reader { return s3Reader{f.bucket} }
func (f s3Factory) NewWriter() Writer { return s3Writer{f.bucket} }
func (f s3Factory) NewLocker() Locker { return dynamoLocker{} }
```

The invariant: **a client that holds one factory cannot obtain a product from another
family.** That impossibility is the entire value; if a client can reach around the factory,
nothing has been bought.

## How not to use it

- **One product in the family.** That is Factory Method with extra ceremony. The pattern
  earns its cost at two products and starts paying at three.

```go
// WRONG — a family interface with one member.
type ReaderFactory interface {
	NewReader() Reader
}
// Symptom: each implementation is one constructor wrapped in a type; a func() Reader says the same.
```

- **Adding a product to the interface.** Every concrete factory breaks at once. The pattern
  is rigid in exactly this direction, and that rigidity is not a bug to be worked around — it
  is the signal that the family boundary was drawn wrongly.

```go
// WRONG — a product that does not vary by family, added to the family.
type StorageFactory interface {
	NewReader() Reader
	NewWriter() Writer
	NewLocker() Locker
	NewMetricsSink() MetricsSink
}
// Symptom: every concrete factory stops compiling, and each fix returns the same sink.
```

- **Using it where products do not actually constrain each other.** If a reader and a writer
  from different families work fine together, the family does not exist and you have built a
  bundle of unrelated factories.

```go
// WRONG — nothing ties the clock to the logger.
type InfraFactory interface {
	NewClock() Clock
	NewLogger() Logger
}
// Symptom: a test wanting a fake clock and the real logger needs a new factory per combination.
```

- **As a configuration mechanism.** Selecting behaviour by swapping a factory at startup is
  legitimate; selecting behaviour per call by choosing a factory is a Strategy.

```go
// WRONG — a whole family chosen per call to make one decision.
func Save(ctx context.Context, obj Object, isDurable bool) error {
	var f StorageFactory = memoryFactory{}
	if isDurable {
		f = s3Factory{bucket: "objects"}
	}
	return f.NewWriter().Write(ctx, obj)
}
// Symptom: the reader and locker are never used; the family is a costly way to pick a writer.
```

**Trade-off:** adding a product is expensive and adding a family is cheap. Confirm that is
the direction your system grows before accepting it.

**Related:** `factory-method.md`, `builder.md`, `../behavioral/strategy.md`
