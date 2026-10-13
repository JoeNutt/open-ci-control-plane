# Template Method

*Behavioral · GoF · Authored for this project*

## What it is

A base class defines the skeleton of an operation as a sequence of calls, some of which are
abstract or overridable. Subclasses supply the varying steps without changing the order or
structure. Inversion of control through inheritance: the base class calls down.

## When to use it

- A sequence is **genuinely invariant** and its steps are not — a build pipeline whose phases
  are fixed, a test harness's setup-exercise-verify-teardown, a report's fetch-transform-render.
- Common steps should live in one place, with specialisations filling the gaps.
- The framework owns the flow and callers own the content, by design.

## How to use it

Make the template method **final** so the sequence cannot be overridden, and keep the hooks
narrow and documented.

```java
abstract class ExtractionRun {
    final Result run(Source src) {            // final: the sequence is the contract
        var raw = read(src);                   // fixed
        var parsed = parse(raw);               // varies
        return summarise(parsed);              // fixed
    }

    protected abstract Parsed parse(Raw raw);  // the single hook
}
```

The invariant: **the template method is not overridable and the hooks do not call back up
into it.** A hook that re-enters the template produces recursion that no one reading the
subclass can see.

## How not to use it

- **A template with many hooks.** Past two or three, the subclass is writing the algorithm
  and the base class is a naming convention. At that point prefer composition — pass the
  varying steps in as strategies.
- **A non-final template method.** If subclasses can override the sequence, there is no
  skeleton, and every subclass may quietly have a different algorithm.
- **Hooks with undocumented ordering or state assumptions.** "Call `parse` only after `read`
  has populated the buffer" is a contract that exists only in the base class's head, and the
  subclass author discovers it by breaking it.
- **Inheriting to reuse the steps rather than to fill them.** If a subclass wants two of the
  five steps and not the flow, it needs the steps as collaborators, not a parent.
- **Deep hierarchies.** Three levels of template method means the effective algorithm is
  assembled from three files and readable from none.
- **Where composition is available and sufficient.** This pattern requires inheritance, which
  is the tightest coupling a language offers. Strategy achieves most of the same ends at arm's
  length. Prefer it unless the base class genuinely owns substantial shared flow.

**Trade-off:** the strongest coupling in the catalogue — the subclass depends on the base
class's internals and its call order, and both are free to surprise the other.

**Related:** `strategy.md`, `../creational/factory-method.md`
