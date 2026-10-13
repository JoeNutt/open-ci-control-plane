# Singleton

*Creational · GoF · Authored for this project*

## What it is

A class guarantees that only one instance of itself exists and provides a global access point
to it. The constructor is made inaccessible and a static accessor returns the single
instance, creating it on first call. The class takes responsibility both for what it does and
for how many of it there are.

## When to use it

Rarely, and the honest answer is **usually not at all** — this is the most criticised pattern
in the catalogue. The narrow cases where it survives:

- The single instance is enforced by something outside the program, not by preference: a
  process-wide lock file, a hardware device with one handle, a registry the operating system
  permits once.
- A language-level idiom provides it with no global state leaking into your design — a Go
  package-level `var` initialised in `init`, a Python module object, a `static final`
  constant holding immutable data.

If the reason is "there should only be one" rather than "there physically cannot be two", the
requirement is **lifetime management**, and a dependency-injection container or an explicitly
passed instance expresses it without the costs below.

## How to use it

If it must exist, make it immutable and make the single-instance guarantee thread-safe at the
language level rather than by hand-rolled locking.

```go
// Package-level, initialised once, immutable after init. No accessor, no lazy branch.
var defaultCapabilities = mustLoadCapabilities()
```

The invariant: **a singleton holding mutable state is shared mutable global state**, and
every argument against global variables applies in full. If it must hold mutable state, it
should not be a singleton.

## How not to use it

This section is the reason the entry exists.

- **As a convenience accessor for a service.** This is the dominant misuse. It couples every
  caller to the concrete class, with the dependency invisible in the signature — a method
  taking no parameters may still reach three subsystems.
- **Anywhere tests matter.** A test cannot substitute the instance, cannot reset it between
  cases, and inherits state from whichever test ran first. Test order becomes significant,
  which is the symptom that reveals the problem.
- **For configuration or a logger, the two most common examples in tutorials.** Both are
  better as a value constructed once at the composition root and passed inward, which is the
  same single instance without the global reach.
- **With lazy initialisation in a concurrent program**, unless the language guarantees it.
  Double-checked locking is famously easy to get subtly wrong.
- **As a cache or registry that accumulates state.** Nothing owns its lifetime, so nothing
  can clear it, and its contents outlive the work that populated them.

**On the "anti-pattern" claim:** it is widely stated — global state, untestable, tightly
coupled, with dependency injection given as the modern replacement
([one summary](https://buraaktasci.medium.com/why-singleton-is-an-anti-pattern-f828c186bd3e),
[another](https://whydoesitsuck.com/why-the-singleton-pattern-sucks-and-you-should-avoid-it/)).
The counter-view is that the underlying idea is sound and the classic implementation is
simply the wrong expression of it in a language with modules
([O'Reilly, on Python](https://www.oreilly.com/library/view/python-real-world-data/9781786465160/ch11s05.html)).
Both are expert opinion rather than settled consensus. The practical position: the costs are
real and well evidenced, so the burden of proof is on the use, not on the objection.

**Trade-off:** there is no version of this pattern that does not make a dependency invisible
at the call site. That is intrinsic, not an implementation flaw.

**Related:** `multiton.md`, `object-pool.md`
