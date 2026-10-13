# Red, green, refactor

*Engineering practice · Derived from the global `tdd` skill — **binding on this project***

## What it is

A three-step cycle repeated once per piece of behaviour. **Red:** write one failing test and
watch it fail. **Green:** write the least production code that makes it pass, however ugly.
**Refactor:** improve the structure of what you just wrote, without changing behaviour, until
it meets the construction standards. All three steps happen every cycle. Commit at green, and
again after refactoring.

## Why it exists

Two separate problems, each solved by a different step.

**Red exists because a test never observed failing proves nothing.** It may assert nothing,
target the wrong thing, or pass for an unrelated reason. Watching the failure — and checking
the message is the one you expected — is the only evidence the test is wired to the behaviour.

**Refactor exists because code at green is unrefactored by construction.** The green step's
instruction is to commit whatever sins are necessary: hardcode, duplicate, take the shortcut.
That is not a failure of discipline, it is the point — it gets to a passing state fast so the
design can be improved with a safety net. Code written under that instruction is, by
definition, not yet code you would keep.

**This is the step that gets dropped, and dropping it is the single most common failure in
agent-written code.** The cycle is misremembered as red → green → done, which leaves every
shortcut from the green step permanently in the codebase. Each one individually looks
tolerable; accumulated across a hundred cycles they are the codebase.

## When to use it

Every time production code that can be tested is written or changed — a feature, a method, an
extension to existing behaviour. **A bug fix starts with a failing test that reproduces the
bug**, because that test is what proves the diagnosis and prevents the regression.

The one legitimate exception is code that genuinely cannot be tested — and that is a design
finding to report, not a licence to skip the cycle.

## How to use it

The full cycle on one piece of behaviour. Note that green is deliberately bad and the
refactor step is what fixes it.

```go
// RIGHT — step 1, RED. One test, run first, watched failing.
func TestBoundsRejectsOversizedChange(t *testing.T) {
    _, err := CheckBounds(Change{Files: 1200}, Limits{MaxFiles: 1000})
    if !errors.Is(err, ErrTooManyFiles) {
        t.Fatalf("want ErrTooManyFiles, got %v", err)
    }
}
// Run it. It fails to compile — that counts as red. Confirm the message is the expected one.
```

```go
// RIGHT — step 2, GREEN. Least code that passes. Hardcoded on purpose.
func CheckBounds(c Change, l Limits) (Change, error) {
    if c.Files > 1000 {               // magic number, deliberately
        return Change{}, ErrTooManyFiles
    }
    return c, nil
}
// Run it. It passes. Every other test still passes. Commit.
```

```go
// RIGHT — step 3, REFACTOR. Same behaviour, standards met.
func CheckBounds(c Change, l Limits) (Change, error) {
    if c.Files > l.MaxFiles {         // the parameter is now actually used
        return Change{}, fmt.Errorf("%w: %d files exceeds %d", ErrTooManyFiles, c.Files, l.MaxFiles)
    }
    return c, nil
}
// Tests stay green throughout. Commit again.
```

**The invariant: the cycle is not complete until a refactor verdict has been stated.** One of
exactly two lines, every time:

- `Refactored: <what changed and why>`
- `No refactor needed: checked <triggers>, none fired.`

The second is legitimate and common. **Producing neither is not.** Run the trigger checklist
*before* forming an opinion about the code — duplication, routine size, nesting, naming,
magic values, conditionals over one value, misplaced logic, untrusted input. Most cycles fire
nothing and cost one scan.

## How not to use it

**Treating the cycle as red → green → done.**

```go
// WRONG — green accepted as finished. The magic number and the swallowed
// context from the green step are now permanent.
func CheckBounds(c Change, l Limits) (Change, error) {
    if c.Files > 1000 {
        return Change{}, ErrTooManyFiles
    }
    return c, nil
}
// Symptom: `l` is an unused parameter and 1000 is unexplained. The test passes,
// so nothing complains. Verdict line absent.
```

**Writing the test after the code.**

```go
// WRONG — the implementation existed first, so the test is shaped to it.
func TestCheckBounds(t *testing.T) {
    c, err := CheckBounds(Change{Files: 5}, Limits{MaxFiles: 1000})
    if err != nil { t.Fatal(err) }
    _ = c
}
// Symptom: it asserts the happy path only, was never seen failing, and would
// still pass if the bound check were deleted entirely.
```

**Writing a suite instead of one test.** Several failing tests at once means you cannot tell
which production change satisfied which, and the green step stops being minimal. One red at a
time; never write the second while the first is red.

**Changing the test to get green.**

```go
// WRONG — the bound was 1000, the code returned 1024, so the test moved.
_, err := CheckBounds(Change{Files: 1024}, Limits{MaxFiles: 1024})
// Symptom: the requirement now documents the implementation. Fix the code, or
// fix a test that was genuinely wrong — and say which you did.
```

Likewise: **never delete or skip a failing test to reach green.** Report it.

**Changing behaviour during the refactor step.** If tests go red while refactoring, behaviour
changed — revert and take a smaller step. Adding a case, fixing a noticed bug and optimising
for speed are all *not* refactoring; the first two are the next red, and tuning comes after
correctness with a measurement.

**Asserting on internals, or mocking heavily to make a test possible.**

```go
// WRONG — asserts a call sequence, so the refactor step is now blocked.
mockRepo.EXPECT().Begin().Times(1)
mockRepo.EXPECT().Save(gomock.Any()).After(begin)
// Symptom: any structural improvement breaks the test without breaking behaviour.
// Prefer a working in-memory fake and assert on the outcome.
```

A test that is hard to write is a design finding. Heavy mocking means too many collaborators
or the wrong ones; needing a clock, database or network to check a business rule means a port
is missing. Fix the direction before reaching for a more powerful mock — see
`../architecture/testability-as-design-feedback.md`.

**Trade-off:** the cycle is slower per line than writing the code directly, and it buys a
suite that makes every later change cheaper. The cost is front-loaded and the benefit is not.

**Related:** `../architecture/testability-as-design-feedback.md`,
`../architecture/incremental-architecture.md`
