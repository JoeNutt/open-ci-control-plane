# Abstract Factory

*Creational · GoF · Authored for this project*

## What it is

An interface declares a creation method for each member of a family of related products.
Each concrete factory implements that interface and returns products from one coherent
family. Client code holds the factory interface and the product interfaces, so it can build
a complete set of collaborating objects without naming any concrete class and without being
able to mix families by accident.

## When to use it

- Products are used **together** and a mismatched combination is a defect — a Windows
  scrollbar inside a GTK window, a Postgres statement builder paired with a MySQL dialect.
- The family is swapped wholesale: per deployment target, per tenant, per supported backend.
- The set of *products in the family* is stable, and the set of *families* is what grows.

## How to use it

Give the factory one method per product, and let the family boundary be the thing that
cannot be crossed.

```go
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
- **Adding a product to the interface.** Every concrete factory breaks at once. The pattern
  is rigid in exactly this direction, and that rigidity is not a bug to be worked around — it
  is the signal that the family boundary was drawn wrongly.
- **Using it where products do not actually constrain each other.** If a reader and a writer
  from different families work fine together, the family does not exist and you have built a
  bundle of unrelated factories.
- **As a configuration mechanism.** Selecting behaviour by swapping a factory at startup is
  legitimate; selecting behaviour per call by choosing a factory is a Strategy.

**Trade-off:** adding a product is expensive and adding a family is cheap. Confirm that is
the direction your system grows before accepting it.

**Related:** `factory-method.md`, `builder.md`, `../behavioral/strategy.md`
