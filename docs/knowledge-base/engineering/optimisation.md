# Optimisation

*Construction · Derived from the global `cc2-construction` skill and this project's own
measurements — **binding on this project***

## What it is

Changing working code to cost less, justified by a measurement of the same workload before and
after. The measurement is the deliverable, not the change: an optimisation with no number
attached is a guess that happens to compile. Go's instrument is `testing.B`, run over a fixture
shaped like the real input rather than a convenient one.

## Why it exists

Because cost intuition is unreliable in both directions, and this project has measured itself
being wrong both ways within one card.

**Too pessimistic.** Classifying a thousand changed paths against seventy-two patterns was
estimated at 780ms by extrapolating a worst-case per-match figure. Measured on a realistic
pattern set it was **76ms** — the estimate was ten times too high because the worst case was not
the common case.

**Too optimistic.** Collapsing those seventy-two patterns into nine combined alternations was
expected to be several times faster, because it does eight times fewer matches. Measured:
**73ms against 81ms**, a nine per cent difference. Each combined match cost 8.1µs where each
single-pattern match cost 1.1µs, because an RE2 program grows with the alternation and the walk
grows with it. Eight times fewer calls, seven times the cost per call, no win.

The second is the more expensive mistake, because the change would have been adopted on
reputation and kept forever.

## When to use it

- A measurement on representative input shows a **stated** budget being breached. This project's
  budgets are numbered requirements — NFR-1's 5ms evaluation p99, NFR-2's 3s end to end — so the
  question "is this too slow?" has an answer that is not an opinion.
- Correctness is already settled and tested. The TDD cycle's refactor step is explicit that
  tuning comes after correctness, with a measurement.
- The alternative has been measured too, not just the current code. A technique's reputation is
  not a measurement of it in this workload.

Not when a number has merely gone up. 76ms of a 3s budget is 2.5 per cent, and spending a day
making it 8ms buys nothing anyone asked for.

## How to use it

Benchmark the real shape of the input, report allocations, and measure the alternative before
adopting it.

```go
// RIGHT — a fixture shaped like the workload, allocations reported, b.Loop for timing.
func BenchmarkKindsOfRealisticChange(b *testing.B) {
    rules := realisticRules(b)  // nine kinds, eight patterns each, as an adopter would write
    paths := thousandPaths()    // NFR-5's bound, not three paths
    b.ReportAllocs()
    for b.Loop() {
        rules.KindsOf(paths)
    }
}
```

```go
// RIGHT — the candidate replacement, benchmarked beside it rather than argued for.
func BenchmarkCombinedAlternationPerKind(b *testing.B) {
    expressions := combinedPerKind()
    paths := thousandPaths()
    b.ReportAllocs()
    for b.Loop() {
        for _, expression := range expressions {
            for _, candidate := range paths {
                expression.MatchString(string(candidate))
            }
        }
    }
}
```

**The invariant: the number goes in the commit message.** A measurement that lives only in a
terminal cannot be disagreed with later, and the next person to wonder whether this is slow has
to measure it again.

Two Go facts worth knowing before reaching for either, because they decide most matching
questions here. There is **no `RegexSet` in the standard library** — the options are a loop over
precompiled expressions, one combined alternation with named groups, or a literal prefilter
(Aho-Corasick) that narrows candidates before running any expression. And `regexp` is **RE2**,
linear in the length of the input whatever the pattern, so the risk of a large combined
expression is compiled program size and match cost, never catastrophic backtracking.

## How not to use it

**Extrapolating from a worst case.**

```go
// WRONG — one adversarial pattern becomes the estimate for all of them.
// worst := 10.9µs  // "**/a/**/x.go" against twenty segments
// estimate := worst * 1000 * 72  // 780ms
// Symptom: the real figure was 76ms, because typical patterns cost a fraction of
// the worst one. The estimate justified an optimisation that was not needed.
```

**Optimising against no stated budget.**

```go
// WRONG — faster than what?
// "76ms felt slow, so I combined the patterns."
// Symptom: nobody can say whether the result is fast enough, so the work has no
// end. Name the requirement first; NFR-2's three seconds makes 76ms a non-issue.
```

**Adopting a technique on reputation.**

```go
// WRONG — fewer calls assumed to mean less work.
expression := regexp.MustCompile("^(?:" + strings.Join(seventyTwoAlternatives, "|") + ")$")
// Symptom: eight times fewer calls at seven times the cost each. Measured at 73ms
// against 81ms. The change adds a translation step, loses which pattern matched,
// and buys nine per cent — which is inside the noise of the thing it replaced.
```

**Recompiling inside the loop.**

```go
// WRONG — the pattern is compiled once per candidate.
for _, candidate := range paths {
    if regexp.MustCompile(source).MatchString(candidate) {  // compiles 1000 times
        matched = append(matched, candidate)
    }
}
// Symptom: the profile is dominated by regexp.Compile rather than by matching.
// Compile once, at parse, and keep the compiled value in the type.
```

**Benchmarking a fixture that is not the workload.** Three paths and one pattern will show any
implementation as instantaneous. Bound the fixture by the requirement — NFR-5's thousand changed
files — so the number means something.

**Trade-off:** measuring properly costs more than changing the code, and most measurements end
in no change at all. That is the point: the two findings above cost one benchmark each and saved
an unnecessary rewrite and a wrong estimate.

**Related:** `tdd-red-green-refactor.md`, `../security/input-validation.md`,
`../architecture/proportionality.md`
