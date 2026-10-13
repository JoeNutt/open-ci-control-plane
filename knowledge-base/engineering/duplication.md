# Duplication

*Construction · Derived from the global `cc2-construction` and `tdd` skills — **binding on this project***

## What it is

The defect is the same **rule** expressed twice, whether or not the text matches. A limit in
the model and again in the handler is duplication. A constant declared in two packages is
duplication. A domain rule transcribed into SQL is duplication. Four kinds worth naming:
identical code, **structural** duplication (same shape, different names), **knowledge**
duplication (same rule, no shared text), and test/production duplication of a magic value.

## Why it exists as a named standard

Because the obvious form — copy-pasted text — is the least dangerous one. It is visible, and a
grep finds it. Knowledge duplication is the kind that rots: the two copies drift, one becomes
authoritative by accident, and nothing in the build notices.

It is also the **primary target of the refactor step** in the TDD cycle, because the green
step's instruction to duplicate freely is what creates it.

## When to use it

Every refactor step, as the first thing scanned for. Exact duplication created while getting
to green goes **now**, not on its third appearance — the rule of three is a guide for
*extracting an abstraction*, not a licence to leave two copies.

And the qualification that decides most real cases: **deduplicate within a boundary, translate
across one.** If the duplicate lives on the other side of a module or service boundary, that is
an architecture question, not a refactor — see `../architecture/dry-across-boundaries.md`.

## How to use it

Give every piece of knowledge exactly one owner inside its boundary.

```go
// RIGHT — the limit is declared once and read by everyone who needs it.
package bounds

const MaxFiles = 1000

func Check(c Change) error {
    if c.Totals.Files > MaxFiles {
        return fmt.Errorf("%w: %d exceeds %d", ErrTooManyFiles, c.Totals.Files, MaxFiles)
    }
    return nil
}
```

```go
// RIGHT — the test reads the same constant. No second copy of the number.
func TestRejectsAtLimit(t *testing.T) {
    err := bounds.Check(Change{Totals: Totals{Files: bounds.MaxFiles + 1}})
    if !errors.Is(err, bounds.ErrTooManyFiles) { t.Fatal(err) }
}
```

**The invariant: knowledge has exactly one owner.** A comment saying "keep this in sync with
X" is an admission that it has none.

## How not to use it

**Knowledge duplication with no shared text.**

```go
// WRONG — the same rule in two places, no copied characters.
// internal/bounds/check.go
if c.Totals.Files > 1000 { return ErrTooManyFiles }

// internal/http/handler.go
if len(req.Files) > 1000 {                       // "fail fast for a better error"
    http.Error(w, "too many files", 413)
    return
}
// Symptom: the limit changes in one place. The other silently becomes the real
// contract, and which one wins depends on call order.
```

**A constant declared twice.**

```go
// WRONG — the same literal in two packages is the strongest form of the defect.
package bounds   ; const MaxFiles = 1000
package telemetry; const maxFiles  = 1000     // "so we don't import bounds"
// Symptom: a grep for 1000 finds two. Nothing finds the relationship.
```

**A magic value repeated between test and production.**

```go
// WRONG — the test hardcodes what the code hardcodes.
if c.Totals.Files > 1000 { ... }               // production
err := Check(Change{Totals: Totals{Files: 1001}})  // test
// Symptom: changing the limit requires changing both, and a test that still
// passes against the old number proves nothing.
```

**Deduplicating across a boundary.** Forcing one shared representation on two independently
released modules couples them, and that usually costs more than the duplication did. Also:
"deduplicating" by inventing a `shared` package is relocating the problem into a module with
no cohesion.

**Treating deliberate mapping as duplication.** The three representations of a thing at a
database row, a domain entity and a wire contract are not accidental copies — they change for
different reasons.

**Trade-off:** removing duplication adds an indirection and a dependency from each former copy
to the new owner. Inside a boundary that is nearly always right; across one it is nearly
always wrong.

**Related:** `../architecture/dry-across-boundaries.md`, `tdd-red-green-refactor.md`, `naming.md`
