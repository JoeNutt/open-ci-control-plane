# Security review and testing

*Security · Derived from the global `sec-review-testing` skill — **binding on this project***

## What it is

A distinct pass with a distinct question: **not "is this correct?" but "how would I attack
this?"** Review by exposure rather than uniformly, in multiple focused passes. Then test that
unintended behaviour is **prevented**, which is a different exercise from testing that
intended behaviour works.

## Why it exists

Because ordinary review and ordinary tests do not find these defects. A missing
authorisation check looks exactly like working code and passes every test of the happy path.
Scanners do not find it either — it is an absence, and there is no pattern for absence.

And because **no single technique catches most defects.** Review, testing, scanning and
fuzzing each catch a partly different set, so a clean result from one is not evidence of
security.

## When to use it

Never review a large codebase uniformly — you will run out of attention before reaching the
risky part. Rank by exposure:

1. Code handling untrusted input — handlers, parsers, deserialisers, uploads, webhooks.
2. Authentication and authorisation — every check, and every path that skips one.
3. Anything crossing a trust boundary.
4. Code running with elevated privilege, or granting it.
5. Cryptography, secrets, session handling.
6. Anything recently changed, especially under time pressure.
7. Code with weak coverage and no owner.

Then **one pass per question** — injection, then authorisation, then secrets. That finds
substantially more than a single undirected read.

## How to use it

Test the denial. This is the part that gets skipped, and it is the part that matters.

```go
// RIGHT — the negative test. Two tenants in the fixture, and the assertion is
// that the stranger is refused.
func TestDecisionNotReadableByOtherTenant(t *testing.T) {
    repo := newRepo(t)
    owned := repo.mustCreate(t, Decision{TenantID: "tenant-a"})

    _, err := repo.DecisionForCaller(ctx, owned.ID, Caller{TenantID: "tenant-b"})

    if !errors.Is(err, ErrNotFound) {
        t.Fatalf("tenant-b read tenant-a's decision: %v", err)
    }
}
```

```go
// RIGHT — hostile input, as a table. Each row is a real bypass attempt.
func TestRejectsHostilePaths(t *testing.T) {
    for _, in := range []string{
        "../../etc/passwd", "....//etc/passwd", "%2e%2e%2fetc%2fpasswd",
        "/absolute/path", "a\x00b", strings.Repeat("a/", 5000),
    } {
        if _, err := resolveInBundle(root, in); err == nil {
            t.Errorf("accepted hostile path %q", in)
        }
    }
}
```

```go
// RIGHT — fuzz the parser. The highest-yield technique for untrusted input.
func FuzzParseEvent(f *testing.F) {
    f.Add([]byte(`{"action":"opened"}`))
    f.Fuzz(func(t *testing.T, b []byte) {
        _, _ = parseEvent(b)       // must not panic, hang or allocate unboundedly
    })
}
```

**The invariant: every access rule has a test proving it refuses**, and every vulnerability
fixed gets a regression test. A security bug without a test is one refactor from returning.

The review questions, for each item: where does the data come from and who controls it; what
happens if it is hostile; what is trusted here and why; what does this assume; **is the check
reachable from every path** — including error paths, retries, background jobs, admin tools and
the second step of a wizard; and what happens when this fails.

## How not to use it

**Testing only the success.**

```go
// WRONG — proves the owner can read. Proves nothing about anyone else.
func TestOwnerCanReadDecision(t *testing.T) {
    d, err := repo.DecisionForCaller(ctx, id, owner)
    if err != nil { t.Fatal(err) }
    if d.ID != id { t.Fatal("wrong decision") }
}
// Symptom: the suite is green with the authorisation check deleted.
```

**Reviewing uniformly, in one pass, looking for everything.** Attention runs out before the
handlers, and a single undirected read finds less than three focused ones.

**Trusting one technique.** "Static analysis is clean" is not evidence; neither is "the tests
pass". They catch different sets, and the dominant class here — a missing check — is caught
by none of them.

**Stopping the trace at the database.**

```go
// WRONG — "it comes from the database, so it is trusted".
reason := row.Reason          // written by an earlier request, unvalidated
renderHTML(w, reason)
// Symptom: stored injection. "It comes from the database" defers the question to
// whoever wrote the row.
```

**Padding the report.** Rank by exposure and impact — anonymous remote code execution
outranks a missing header, however many headers are missing. **Say clearly when you found
nothing significant**; padding trains people to skim the next one. And be explicit about what
the review could not cover.

**Trade-off:** multiple focused passes cost more time than one read, and fuzzing costs CPU
and produces failures that are tedious to triage. Both find defects that no amount of careful
reading does.

**Related:** `threat-modeling.md`, `authorization.md`,
`../engineering/tdd-red-green-refactor.md`
