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

## You may not end a turn without a refactor verdict

State exactly one of these lines. It is checked mechanically, so the wording matters:

```
Refactored: <what changed and why>
```

```
No refactor needed: checked <which questions>, none fired.
```

The second is legitimate and common. **Producing neither is not**, and a `Stop` hook will
refuse to let the turn end until one is present.

## Also non-negotiable

- **Never change a test to make it pass.** Fix the code, or fix a test that was genuinely
  wrong — and say which you did and why.
- **Never delete or skip a failing test to get green.** Report it.
- **Test the denial, not just the success.** Every control needs a test proving it refuses.
- Not refactoring, and not allowed in this step: adding a case, fixing a noticed bug,
  optimising for speed. The first two are the next RED; tuning comes after correctness, with a
  measurement.
- **Strip the comments you wrote during red and green.** Green permits anything, comments
  included — so scratch notes, narration and reminders are expected to exist by the time you
  get here, and removing them is part of this step. What survives: a doc comment on each
  exported name in the Google Go form, complete sentences beginning with the name; and a
  constraint whose reason lives outside the code, with its reference; and the
  `// SPDX-License-Identifier: GPL-3.0-or-later` first line, which is a legal marker and is
  never stripped. Nothing else.
  Self-justification is a defect, not a courtesy: `knowledge-base/engineering/comments.md`.
