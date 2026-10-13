# Comments

*Construction · Derived from the global `cc2-construction` skill, with doc comment form from
the [Google Go Style Guide](https://google.github.io/styleguide/go/decisions) — **binding on
this project***

## What it is

Two standards, covering two different halves, because neither covers both.

**Form — Google Go Style Guide.** Every top-level exported name carries a doc comment.
Unexported names carry one only when their behaviour or meaning is not obvious. A doc comment
is **complete sentences beginning with the name of the thing it describes**, optionally
preceded by an article, written for people *using* the package rather than implementing it.

**Quantity — this project's own standard.** Default to no comments. Google is explicit about
doc comment form and **silent on what not to comment** — no rule on narration, redundancy or
commented-out code. That half is governed here, and it is strict: besides doc comments, the
only permitted comment is a **recorded external constraint** with its reference.

## Why it exists

Comments are not compiled, not tested, and not checked by anything, so they drift and become
confidently wrong. **A stale comment is worse than none, because it is trusted.**

The quantity rule exists because the dominant failure in generated code is **narrating its own
reasoning**. It reads as helpful and is noise: it explains a visible decision to a reader who
can see the code, assumes that reader knows less than they do, and will be wrong within two
refactors. A comment is usually a failure to express something in code — and the fix is a
better name or a smaller function, which puts the explanation somewhere the compiler checks.

## When to use it

**A doc comment** on every exported top-level name, and on an unexported one only where the
behaviour is not obvious from the name and signature.

**A constraint comment** when the reason lives *outside* the code: a vendor quirk, a spec
requirement, an ordering imposed from elsewhere. The test: **is the reason outside the code, or
inside your head?** A vendor returning 200 on failure is outside — no reader could deduce it.
"I chose a map for speed" is inside.

**An SPDX licence identifier** as the first line of a source file. This project is
GPL-3.0-or-later, and the identifier is a legal marker rather than a comment:

```go
// SPDX-License-Identifier: GPL-3.0-or-later
```

It is **never** stripped during a refactor. Nothing else qualifies.

## How to use it

```go
// RIGHT — package comment directly above the clause, no blank line. One per package.
// Package policy evaluates structural facts about a change against declarative rules.
package policy
```

```go
// RIGHT — begins with the name, complete sentences, written for the caller.
// CheckBounds verifies a change against the configured limits and returns it as a
// Bounded value.
//
// It returns ErrTooManyFiles, ErrPatchTooLarge or ErrDepthExceeded naming the limit
// that was breached. The change is never truncated.
func CheckBounds(c Change, l Limits) (Bounded, error)
```

```go
// RIGHT — struct field comments may be short phrases taking the field as the subject.
type Limits struct {
	MaxFiles      int // changed files accepted before the change is refused
	MaxPatchBytes int // per-file patch size
}
```

```go
// RIGHT — a recorded external constraint, with its reference.
// GitHub returns 200 with an empty file list when the diff is too large to compute,
// rather than an error. SUPPORT-4471.
if resp.StatusCode == http.StatusOK && len(files) == 0 && pr.Changes > 0 {
	return ErrDiffUnavailable
}
```

**The invariant: if code needs a comment to be understood, rename or extract first.** Comment
only when that genuinely fails.

Two placement rules worth knowing: a package comment sits **directly above the package
clause** with no blank line and appears in exactly one file; and a maintainer-facing comment
about the whole file goes **after the import declarations**, where it is not a package comment
and not subject to its rules.

## How not to use it

**Narrating the next line.**

```go
// WRONG
// loop through the changed files
for _, f := range c.Files {
	// increment the counter
	total++
}
// Symptom: the comment restates the code. Delete both.
```

**Justifying your own choices — the dominant failure.**

```go
// WRONG — four self-commentaries on visible decisions.
// using a map here for O(1) lookup
kinds := map[string]bool{}
// this handles the edge case where the list is empty
if len(c.Paths) == 0 { return Verdict{} }
// we validate first so the later code can trust it
bounded, err := CheckBounds(c, l)
// helper function to format the path
func fmtPath(p string) string
// Symptom: the map, the guard and the order are all visible. fmtPath should say
// it in its name. None of these reasons is outside the code.
```

**A doc comment that does not begin with the name, or carries rationale.**

```go
// WRONG — breaks the form, then adds history and implementation notes.
// This decides things. It was split out of Evaluate during the refactor because
// Evaluate was getting long. Uses a map for performance.
func (g Gate) Decide(c Change) Verdict
// Symptom: Godoc renders a sentence that does not name the symbol. Rationale and
// history belong in the commit message, where they are dated and attributed.
```

**Banners, logs, tags and commented-out code.**

```go
// WRONG
// ===================== HELPERS =====================
// Changed by J. Nutt 2026-03-01
// TODO: refactor this later
// if oldPath != "" { return oldPath }   // keeping just in case
// Symptom: a file needing signposting needs splitting; the date and author are
// version control's job; the TODO is untracked and permanent; the dead code is
// already in the history.
```

**Explaining a language or library feature.** A comment telling the reader what a map, a
channel, a mutex or an `errgroup` does assumes they do not know the language. They do.

**Using the rule as licence to leave things unexplained.** It targets noise, not context.
Where a reader genuinely needs information that cannot live in code, withholding it is also a
failure — record it, briefly, as a constraint with its reference.

**Trade-off:** the discipline pushes work into naming and decomposition, which is slower than
writing a sentence. What it buys is documentation that cannot go stale.

**Related:** `naming.md`, `code-layout.md`
