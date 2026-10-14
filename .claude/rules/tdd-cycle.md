---
paths:
  - "**/*.go"
  - "**/*.rego"
---

# The cycle is red, green, refactor — three steps, every time

This applies to **you**, whether you are the main session or a subagent. A subagent does not
inherit a conversation, so everything needed is here.

**The failure this rule exists to prevent:** running red → green → done, and leaving the
green step's shortcuts in the codebase permanently. Code at green is unrefactored *by
construction* — the green step's instruction is to commit whatever sins are necessary — so
code written under it is by definition not code to keep.

---

## RED — a test must fail before anything else exists

- Write **one** test for the next small piece of behaviour. Not a suite. Never write a second
  while the first is red.
- **The size test, because "one test" gets stretched:** if the test cannot compile until more
  than **one** new named thing exists, the increment is too big. Split it and start with
  whichever piece can be tested alone.

  `RED: the port, the assembler and the fake` is three cycles announced as one. So is any RED
  that introduces an interface and an implementation together. Find the piece that is a pure
  function over values you already have, and start there.
- **One commit at green introduces one new named thing.** That is how the increment size is
  reviewable after the fact, and it is checked by reading the diff.
- **Run it. Watch it fail.** Confirm the failure message is the one you expected. Failing to
  compile counts as failing. A test never observed failing proves nothing — it may assert
  nothing, or pass for an unrelated reason.
- **No production code exists before a failing test that demands it.** If asked to write code
  with no test, write the test first, or state plainly that the code is untestable — which is
  a design finding, not a licence to skip.
- A bug fix starts with a failing test that reproduces the bug.

## GREEN — make it pass, by any means. Standards are suspended.

- Write **no more** production code than passes the currently failing test.
- **Rules be damned.** Hardcode the value. Duplicate the block. Ignore naming. Skip the
  abstraction. Leave the magic number. Speed over elegance, deliberately.
- Three routes: *fake it* (return the constant, generalise later), *obvious implementation*
  (just write it, when it truly is obvious), *triangulation* (add a second test that forces
  the generalisation).
- Run the test. It passes, and every other test still passes. **Commit.**

This step is the only place in the project where the standards do not apply. That is not
sloppiness — it is what makes the refactor step have something to do.

## REFACTOR — now reason against the knowledge base

Mandatory. Structure changes; **observable behaviour does not**. Tests stay green throughout —
if they go red you changed behaviour, so revert and take a smaller step. Refactor the new
code, the old code it touched, **and the tests**.

Work the five questions. Each is a real question to reason about, not a box:

| Question | Read |
|---|---|
| **Can a design pattern express this better than what I wrote?** | `knowledge-base/design-patterns/README.md`, then the one entry |
| **Are the architectural boundaries right?** Dependency direction inward, ports owned by the consumer, no vendor type across a boundary, no layer-skipping | `knowledge-base/architecture/README.md` |
| **Is this clean code?** Naming, one job per routine, parameter count, nesting depth, comments, duplication, a conditional chain that is really a table | `knowledge-base/engineering/README.md` |
| **Is hexagonal design applied properly, or just invoked?** A port is an interface the *inside* declares; an adapter translates and carries no business rule | `knowledge-base/architecture/ports-and-adapters.md` |
| **Does this defend against the OWASP Top 10?** Untrusted input, authorisation on the object, injection, secrets, error paths that fail open | `knowledge-base/security/owasp-top-10/README.md` |

Run the questions **before** forming an opinion about the code. Your familiarity with
something you wrote sixty seconds ago is not evidence of its quality.

Take **one** refactoring at a time, tests green between each. If several things fire, take the
most costly first. A refactor too large for this cycle does not get skipped — finish the
cycle, record what it is in the terms of the standard it violates, and do it as its own change
from green.

Then **commit again.** Two clean commits beat one mixed one.

---

## One card at a time

Work comes from the board: https://github.com/users/JoeNutt/projects/2

**Take the Ready card with the lowest `Sequence`, and only that card.** The board is ordered
so card N depends only on cards before it; never pick a card further down because it looks
smaller, more interesting or unblocked. Cards with no `Sequence` are not yet ordered and are
not work. Below the sequenced cards the board is kept in priority order — every Must, then
every Should, then every Could; within each, by phase, then card number — so the next batch to
sequence is always the top of what remains. When the sequenced cards run out, the next batch is sequenced from the SAD — every
obligation a card carries checked for the card that produces its data — before any of it is
started.

**Hold exactly one card in In Progress.** Claim it by setting both the `Agent` and `Status`
fields in the same call, work it through the cycle, and move it to `Review` or `Blocked`
before claiming another. When a card closes, set the next `Sequence` card to `Ready`.

Nothing enforces this. It is a convention, and a breach is visible on the board.

**A card closes on proof, and nothing else starts until it does.** Not "the code works", not
"the tests pass" — proof that every obligation in the card's checklist is met, quoted from the
specification and answered one at a time. The checklist is the requirement's text **and** its
user story's acceptance criteria **and** the §13 contract and §14 state rows that describe its
output. A requirement with three clauses needs three answers. Partial delivery is reported as
partial and the card stays open. Never close a card by moving an unmet obligation to another
card at closing time — card #8 was closed that way and had to be reopened. Scope moves happen
before work starts, written on both cards.

Merging is not closing. Five pull requests merged into card #2 without closing it, because
one clause of FR-3 reaches into another card — which is a scoping defect to surface, not to
work around by starting something else. If a card cannot close from inside its own scope,
say so and fix the card before writing code.

`Won't` cards are not work and are not on the board. They are closed as not planned, because
the specification already records them: §9's table carries each with its rationale, §18
excludes them from the estimate base, and the "Won't Have, this time" section states why each
is out. Bringing one into scope means reopening it deliberately.

---

## Every cycle ends with a refactor verdict, in that cycle's commit

Once per cycle, in the commit message, attached to the diff it describes. Not once per turn:
a single verdict covering several cycles is a verdict about none of them. State one of:

```
Refactored: <what changed, and which entry demanded it>
```

```
No refactor needed: checked <all five, each naming the entry read and what it says>, none fired.
```

**Cite the entry and what it says. A verdict naming only a topic is not a check.** "boundaries
unchanged" is unfalsifiable — a conclusion with no evidence that anything was read, which is
exactly what a verdict written from memory looks like. A real one reads: *coupling.md says a
Bounded can only exist after bounds passed and that Evaluate cannot be called with Raw;
ParseDocument takes a raw []File, so it does not have that shape.* A reader can open the file
and disagree with that.

**Name the five questions by their own names** — design pattern, architectural boundaries,
clean code, hexagonal, OWASP. Naming five items from inside one of them is one check wearing
five hats: duplication, routine size, nesting, naming and comments all belong to question 3,
and a verdict listing those leaves patterns, boundaries and OWASP unexamined.

**Answering from memory is the failure this step exists to prevent.** The entries are the
specification; your recollection of them is not, and the reason they were written down is that
recollection produces plausible code the standards reject. If a verdict did not involve opening
a file, it did not happen.

## Also non-negotiable

- **Never change a test to make it pass.** Fix the code, or fix a test that was genuinely
  wrong — and say which you did and why.
- **Never delete or skip a failing test to get green.** Report it.
- **Test the denial, not just the success.** Every control needs a test proving it refuses.
- Not refactoring, and not allowed in this step: adding a case, fixing a noticed bug,
  optimising for speed. A bug or missing case noticed here becomes **the next RED — the one
  immediately after this cycle, not a later one.** Deferring it to "a future cycle" and
  starting unrelated work is how a security finding stays open across three cycles while every
  verdict faithfully reports it as known. Tuning comes after correctness, with a measurement.
- **Strip the comments you wrote during red and green.** Green permits anything, comments
  included — so scratch notes, narration and reminders are expected to exist by the time you
  get here, and removing them is part of this step. What survives: a doc comment on each
  exported name in the Google Go form, complete sentences beginning with the name; and a
  constraint whose reason lives outside the code, with its reference; and the
  `// SPDX-License-Identifier: GPL-3.0-or-later` first line, which is a legal marker and is
  never stripped. Nothing else.
- **Commit and the tests must be green.** A `.githooks/pre-commit` hook refuses a red suite,
  unformatted files and `go vet` failures. It binds everyone, not only agents.
  Self-justification is a defect, not a courtesy: `knowledge-base/engineering/comments.md`.
