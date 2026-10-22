# Builder

*Creational · GoF · Authored for this project*

## What it is

Construction is separated from representation: a builder exposes steps that accumulate
configuration, and a final call produces the object. GoF's form adds a director that drives
a fixed sequence of steps so the same process can produce different representations. The
common modern form is the fluent builder, which keeps the step separation and drops the
director.

## Why it exists

Once an object has many parts, most of them optional, a constructor taking every part
positionally is unreadable and easy to misorder, and the usual escape — construct, then set
fields — exposes the object half-built, before its cross-field rules hold. The builder gives
the half-built state its own type, so assembly can be gradual while validation happens once,
at the end, before any caller holds the product.

## When to use it

- An object has many optional parts, and constructor overloads have become a combinatorial
  set nobody can read.
- Construction has required ordering or cross-field validation that must happen once, at the
  end, not per setter.
- The same construction sequence must yield different representations — the same parse driving
  a DOM tree or a flat event stream.
- The result should be **immutable**, so it cannot be assembled by mutating it after creation.

## How to use it

Validate in `Build`, not in the setters, and return the built object as immutable.

```go
// RIGHT — setters accumulate; only Build validates and produces the product.
type ServerBuilder struct{ cfg serverConfig }

func (b *ServerBuilder) Addr(a string) *ServerBuilder           { b.cfg.addr = a; return b }
func (b *ServerBuilder) Timeout(d time.Duration) *ServerBuilder { b.cfg.timeout = d; return b }

func (b *ServerBuilder) Build() (*Server, error) {
	if b.cfg.addr == "" {
		return nil, errors.New("addr is required")
	}
	return &Server{cfg: b.cfg}, nil // immutable from here
}
```

The invariant: **a half-configured builder can never be mistaken for the product.** They are
different types, and only `Build` crosses the boundary.

## How not to use it

- **A builder that returns an object which can still be mutated.** Then it is a
  configuration bag with a fluent accent, and the validation in `Build` guarantees nothing
  about the object's later state.

```go
// WRONG — the product can be changed after Build validated it.
func (s *Server) SetTimeout(d time.Duration) { s.cfg.timeout = d }
// Symptom: a negative timeout set after Build reaches the listener; Build never saw it.
```

- **Validating per setter.** `Timeout()` cannot know whether the combination is legal; only
  the terminal call can. Early validation produces rules that are individually true and
  jointly wrong.

```go
// WRONG — a rule about two fields, checked in a setter that sees one.
func (b *ServerBuilder) Timeout(d time.Duration) *ServerBuilder {
	if d > b.cfg.idleTimeout {
		panic("timeout exceeds idle timeout")
	}
	b.cfg.timeout = d
	return b
}
// Symptom: one configuration is accepted or refused depending on the order of the setter calls.
```

- **A builder for three fields, two of them required.** A constructor is clearer. The
  pattern pays for itself with many optional parts, not with any optional part.

```go
// WRONG — required parts expressed as optional steps.
srv, err := new(ServerBuilder).Addr(":8080").Name("gate").Build()
// Symptom: forgetting Addr is a runtime error; NewServer(addr, name) would make it a compile error.
```

- **A `Build` that cannot fail.** If construction has invariants, the terminal call must be
  able to refuse. One that returns the product unconditionally has moved the failure to first
  use, far from its cause.

```go
// WRONG — the terminal call has no way to refuse.
func (b *ServerBuilder) Build() *Server { return &Server{cfg: b.cfg} }
// Symptom: a missing address is found when the server first listens, far from the code that forgot it.
```

- **Where named arguments already are the pattern.** Languages with named or default
  arguments have it built in. Go's nearest equivalent is a keyed struct literal passed to one
  validating constructor; a builder that mirrors the struct field for field adds a type and no
  check.

```go
// WRONG — every step maps to one field of a struct the caller could have written.
cfg, err := new(ConfigBuilder).Addr(":8080").Timeout(5 * time.Second).Build()
// Symptom: the builder repeats ServerConfig{Addr: ":8080", Timeout: 5 * time.Second} and adds nothing.
```

**Trade-off:** a second type to maintain alongside the product, and a construction path the
compiler checks less thoroughly than a constructor signature would.

**Related:** `abstract-factory.md`, `prototype.md`
