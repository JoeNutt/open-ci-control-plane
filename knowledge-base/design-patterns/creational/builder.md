# Builder

*Creational · GoF · Authored for this project*

## What it is

Construction is separated from representation: a builder exposes steps that accumulate
configuration, and a final call produces the object. GoF's form adds a director that drives
a fixed sequence of steps so the same process can produce different representations. The
common modern form is the fluent builder, which keeps the step separation and drops the
director.

## When to use it

- An object has many optional parts, and constructor overloads have become a combinatorial
  set nobody can read.
- Construction has required ordering or cross-field validation that must happen once, at the
  end, not per setter.
- The same construction sequence must yield different representations — the same parse driving
  a DOM tree or a flat event stream.
- The result should be **immutable**, so it cannot be assembled by mutating it after creation.

## How to use it

Validate in `build()`, not in the setters, and return the built object as immutable.

```go
type ServerBuilder struct{ cfg serverConfig }

func (b *ServerBuilder) Addr(a string) *ServerBuilder      { b.cfg.addr = a; return b }
func (b *ServerBuilder) Timeout(d time.Duration) *ServerBuilder { b.cfg.timeout = d; return b }

func (b *ServerBuilder) Build() (*Server, error) {
    if b.cfg.addr == "" {
        return nil, errors.New("addr is required")
    }
    return &Server{cfg: b.cfg}, nil   // immutable from here
}
```

The invariant: **a half-configured builder can never be mistaken for the product.** They are
different types, and only `Build` crosses the boundary.

## How not to use it

- **A builder that returns an object which can still be mutated.** Then it is a
  configuration bag with a fluent accent, and the validation in `build()` guarantees nothing
  about the object's later state.
- **Validating per setter.** `Timeout()` cannot know whether the combination is legal; only
  the terminal call can. Early validation produces rules that are individually true and
  jointly wrong.
- **A builder for three fields, two of them required.** A constructor is clearer. The
  pattern pays for itself with many optional parts, not with any optional part.
- **A `build()` that cannot fail.** If construction has invariants, the terminal call must be
  able to refuse. One that returns the product unconditionally has moved the failure to first
  use, far from its cause.
- **In languages with named or default arguments, reaching for it at all.** Keyword arguments
  are the pattern, built into the language.

**Trade-off:** a second type to maintain alongside the product, and a construction path the
compiler checks less thoroughly than a constructor signature would.

**Related:** `abstract-factory.md`, `prototype.md`
