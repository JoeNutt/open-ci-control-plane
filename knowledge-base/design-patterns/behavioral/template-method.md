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

Go has no inheritance, so there is no base class to call down from, and embedding does not
supply one: a promoted method never dispatches to the type that embeds it. The Go form is a
plain function that owns the sequence and takes the varying step as an interface or func
value. A function cannot be overridden, so the template is final by construction — and the
pattern converges on Strategy, with the function as the context.

```go
// RIGHT — the function is the sequence; the one hook is passed in.
type Parser interface {
    Parse(raw Raw) (Parsed, error)
}

func RunExtraction(src Source, p Parser) (Result, error) {
    raw, err := read(src) // fixed
    if err != nil {
        return Result{}, err
    }
    parsed, err := p.Parse(raw) // varies
    if err != nil {
        return Result{}, err
    }
    return summarise(parsed), nil // fixed
}
```

The invariant: **the template method is not overridable and the hooks do not call back up
into it.** A hook that re-enters the template produces recursion that no one reading the
subclass can see.

## How not to use it

- **A template with many hooks.** Past two or three, the subclass is writing the algorithm
  and the base class is a naming convention. At that point prefer composition — pass the
  varying steps in as strategies.

```go
// WRONG — so many hooks that the implementer writes the algorithm.
type ExtractionHooks interface {
    BeforeRead(src Source)
    Read(src Source) (Raw, error)
    Parse(raw Raw) (Parsed, error)
    Filter(p Parsed) Parsed
    Summarise(p Parsed) Result
}
// Symptom: RunExtraction is a list of calls, and every behaviour lives in the implementations.
```

- **A non-final template method.** If subclasses can override the sequence, there is no
  skeleton, and every subclass may quietly have a different algorithm.

```go
// WRONG — the sequence itself is a replaceable field.
type ExtractionRun struct {
    Run   func(src Source) (Result, error)
    Parse func(raw Raw) (Parsed, error)
}
// Symptom: any caller can replace Run, and two runs under one name follow different algorithms.
```

- **Hooks with undocumented ordering or state assumptions.** "Call `parse` only after `read`
  has populated the buffer" is a contract that exists only in the base class's head, and the
  subclass author discovers it by breaking it.

```go
// WRONG — Parse ignores its argument and reads state an earlier step left behind.
func (p *csvParser) Parse(Raw) (Parsed, error) {
    return parseRows(p.buffer)
}
// Symptom: Parse called before anything fills p.buffer returns no rows and no error.
```

- **Inheriting to reuse the steps rather than to fill them.** If a subclass wants two of the
  five steps and not the flow, it needs the steps as collaborators, not a parent.

```go
// WRONG — embedding the run to borrow two of its steps, not its flow.
type Linter struct{ ExtractionRun }

func (l Linter) Lint(src Source) Findings {
    return lint(l.normalise(l.read(src)))
}
// Symptom: Linter changes whenever ExtractionRun does, though it never uses the sequence.
```

- **Deep hierarchies.** Three levels of template method means the effective algorithm is
  assembled from three files and readable from none.

```go
// WRONG — three levels of embedding, each promoting or shadowing steps.
type BaseRun struct{}
type GoRun struct{ BaseRun }
type GoModuleRun struct{ GoRun }
// Symptom: which parse a GoModuleRun uses depends on promotion depth across three files.
```

- **Where composition is available and sufficient.** This pattern requires inheritance, which
  is the tightest coupling a language offers. Strategy achieves most of the same ends at arm's
  length. Prefer it unless the base class genuinely owns substantial shared flow.

```go
// WRONG — inheritance emulated: the base holds an interface back to its own outer type.
type baseRun struct{ self Parser }

type csvRun struct{ baseRun }

run := &csvRun{}
run.self = run
// Symptom: a reference cycle and a construction-order trap, for what passing a Parser gives.
```

**Trade-off:** the strongest coupling in the catalogue — the subclass depends on the base
class's internals and its call order, and both are free to surprise the other.

**Related:** `strategy.md`, `../creational/factory-method.md`
