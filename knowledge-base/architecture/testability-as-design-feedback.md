# Testability as design feedback

*Architecture · Derived from the global `architecture` skill — **binding on this project***

## What it is

Treating the difficulty of writing a test as information about the design rather than as a
testing problem. **If tests are easy to write, the design is probably good; if they are hard,
it is probably not.** Taken literally, the symptom tells you which structural fault produced
it.

## Why it exists

Because it is the most reliable feedback available on a design, and it arrives continuously
and for free. Every other signal — review, metrics, "it feels wrong" — is slower, rarer or
subjective. A test that is painful to write is an objective, repeatable measurement taken at
the moment the design decision is still cheap to reverse.

It exists as a stated principle because the natural response to the symptom is the wrong one:
reach for a more powerful mock, a test container, a fixture framework. That treats the
measurement as the fault.

## When to use it

Every time a test is hard to write. The specific signals and what each one says:

| Symptom | What it says |
|---|---|
| Heavy mocking needed | Too many collaborators, or the wrong ones |
| Needs a database, clock or network to test a rule | Infrastructure reached the domain; a port is missing |
| Setup longer than the assertion | The unit does too much |
| Test asserts on internals | Behaviour is not exposed at the right level |
| Cannot test without starting the whole app | No seams; everything is wired concretely |
| A private function you want to test directly | It probably wants to be its own unit |

## How to use it

Fix the direction, not the test. The usual fix is a missing port.

```go
// RIGHT — the non-deterministic dependency is behind a port, so the rule is
// testable with no infrastructure and no mock framework.
type Clock interface{ Now() time.Time }

func (g Gate) Decide(c Change, now time.Time) Verdict { /* pure */ }
```

```go
// RIGHT — the test is three lines because the design allowed it to be.
func TestExpiredGrantIsRefused(t *testing.T) {
    g := Grant{expiresAt: time.Unix(100, 0)}
    if g.IsLive(time.Unix(200, 0)) {
        t.Fatal("expired grant reported live")
    }
}
```

Prefer **fakes over mocks** for ports — a working in-memory implementation tests behaviour,
where a mock asserting a call sequence tests your current implementation and blocks the
refactor step.

```go
// RIGHT — a fake. Tests outcomes, survives refactoring.
type fakeRecorder struct{ saved []Verdict }
func (f *fakeRecorder) Save(_ context.Context, v Verdict) error {
    f.saved = append(f.saved, v); return nil
}
```

**The invariant: a business rule is testable with no infrastructure at all.** If it is not,
a dependency points the wrong way.

## How not to use it

**Answering the symptom with better tooling.**

```go
// WRONG — the rule needs a database, so the test starts one.
func TestGateBlocksAuthChange(t *testing.T) {
    db := testcontainers.StartPostgres(t)      // 4s, flaky, needs Docker
    defer db.Stop()
    gate := NewGate(sqlRepo{db})
    // ...
}
// Symptom: a pure decision now depends on Docker. The container made the test
// possible and left the design fault in place.
```

**Mocking a call sequence.**

```go
// WRONG — asserts HOW, so any structural improvement fails the test.
mockSrc.EXPECT().HeadSHA(gomock.Any(), gomock.Any()).Times(1)
mockSrc.EXPECT().FilesChanged(gomock.Any(), gomock.Any()).After(headSHA)
// Symptom: reordering two independent calls breaks a test without breaking
// behaviour. The refactor step is now blocked, which is the step that matters.
```

**Reaching for the clock directly.**

```go
// WRONG — time.Now() inside the rule.
func (g Grant) IsLive() bool { return time.Now().Before(g.expiresAt) }
// Symptom: testing the boundary means sleeping, or monkey-patching the clock.
// Pass the instant in, or put it behind a Clock port.
```

**Exporting internals so a test can reach them.** If the test needs a private function, the
answer is usually that the function is its own unit with its own public behaviour — not that
the test should be allowed in. Widening visibility for a test makes the test the reason the
API is shaped that way.

**Trade-off:** designing for testability means more seams, and seams are indirection. The
compensation is that the properties it produces — modularity, cohesion, separation of
concerns, loose coupling — are the same ones you wanted anyway, viewed from another angle.

**Related:** `../engineering/tdd-red-green-refactor.md`, `ports-and-adapters.md`, `coupling.md`
