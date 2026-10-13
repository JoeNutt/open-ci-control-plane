# Data and scope

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

Declare and initialise a variable immediately before its first use, never at the top of a
long block. Keep scope as narrow as the language permits, and minimise the live span between
first and last reference. Avoid global mutable state; where unavoidable, mediate every access
through accessors. Prefer integers to floats where rounding error accumulates.

## Why it exists

Because the span between a variable's declaration and its last use is the window in which a
reader must keep it in mind, and in which other code can interfere with it. A variable
declared forty lines early is forty lines of obligation for no benefit.

Global mutable state is worse than wide scope: it has no owner, no discipline about who writes
it, and produces action at a distance — a change here alters behaviour there, with nothing in
either place saying so.

## When to use it

Every declaration. Also when reviewing a long routine: variables declared at the top in a
block are usually the symptom of a routine doing several things, each of which wanted its own
scope.

The float rule applies to money, counters, and anything iterated — any quantity where errors
accumulate rather than cancel.

## How to use it

```go
// RIGHT — each value declared where it is first needed, narrowest scope available.
func summarise(c Change) Summary {
    kinds := classify(c.Paths)

    if kinds.Has(KindIaC) {
        resources := parseResources(c)        // scoped to the branch that needs it
        return Summary{Kinds: kinds, Resources: resources}
    }

    return Summary{Kinds: kinds}
}
```

```go
// RIGHT — money in minor units as an integer. No accumulating error.
type Money struct {
    Minor    int64        // pence, cents
    Currency Currency
}
```

```go
// RIGHT — package-level state is immutable and initialised once.
var defaultCapabilities = mustLoadCapabilities()
```

**The invariant: scope is the smallest the language allows.** If a value is used in one
branch, it is declared in that branch.

## How not to use it

**Declaring everything at the top.**

```go
// WRONG — four variables held in mind for the whole routine.
func process(c Change) (Summary, error) {
    var kinds KindSet
    var resources []Resource
    var err error
    var total int
    // ...thirty lines, during which each of these may or may not be set...
}
// Symptom: a reader cannot tell which are live at any point, and the zero value
// of each is a plausible-looking wrong answer if a path forgets to set it.
```

**Mutable package-level state.**

```go
// WRONG — no owner, written from anywhere, racy.
var currentRevision string

func activate(rev string) { currentRevision = rev }
func Decide(c Change) Verdict {
    return Verdict{Revision: currentRevision}     // read from somewhere else entirely
}
// Symptom: two goroutines, one data race, and no signature mentions the coupling.
// Hold it in a type and pass the type.
```

**Floats for money.**

```go
// WRONG — accumulates error, and comparisons stop working.
var minutesSaved float64
minutesSaved += 0.1                          // 0.1 is not representable
if minutesSaved == 0.3 { /* never true after three additions */ }
// Symptom: totals drift and equality fails. Use an integer in minor units.
```

**Reusing one variable for two purposes.** A name that means one thing at the top of a routine
and another at the bottom is two variables sharing storage, and the reader has to notice the
transition.

**Trade-off:** narrow scope sometimes means recomputing or re-declaring a value in two
branches rather than hoisting it. That duplication is usually cheaper than the extended live
span — but see `duplication.md` if it becomes knowledge duplication rather than a repeated
expression.

**Related:** `routines.md`, `control-flow.md`, `duplication.md`
