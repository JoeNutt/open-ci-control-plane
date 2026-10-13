# Knowledge base entry format

Every entry follows the same shape. The point is that a session opens one file, reads five
sections, and knows both what to do and what not to do — without reading its neighbours and
without guessing which part is advice and which is a rule.

## The five required sections, in this order

| # | Heading | Holds |
|---|---|---|
| 1 | `## What it is` | Two to four sentences. The mechanism, not the motivation. A reader who has never heard the name should finish this able to recognise it in code. |
| 2 | `## Why it exists` | The problem it was invented to solve, and what goes wrong without it. The *force* it resolves — not its benefits in the abstract. |
| 3 | `## When to use it` | The conditions that make it the right choice **here**. Written as situations, not benefits: "the set of concrete types is fixed and known at compile time", never "improves flexibility". |
| 4 | `## How to use it` | The correct application, **with a Go code example**, plus the invariant that must hold. |
| 5 | `## How not to use it` | The misuses, **each with a Go code example of the wrong version**, the symptom that reveals it, and the cheaper thing to do instead. |

Sections 2 and 3 are different questions and both are required. *Why it exists* is about the
problem in general; *when to use it* is about recognising that problem in front of you.

Two short optional closers may follow:

- `**Trade-off:**` one sentence naming what the technique costs when applied **correctly**.
- `**Related:**` links to sibling entries, so a reader routes sideways rather than reading a
  whole directory.

## Code examples: the two hard rules

**1. Go, and only Go.** This is a Go project. An example in Java, Kotlin, TypeScript, C# or
Python is noise here — the reader has to translate it before they can use it, and the idiom
often does not survive the translation. Where a pattern is shaped by a language feature Go
lacks, say so in prose rather than reaching for another language.

**2. Every misuse in section 5 carries the wrong code, not just a description of it.** Prose
alone does not work: an agent reading "do not let the adapter leak vendor types" will agree
and then do it anyway, because it never saw what that looks like. Show the broken version,
marked, then the symptom.

```go
// WRONG — the vendor type crossed the port.
type ChangeSource interface {
    FilesChanged(ctx context.Context, pr *github.PullRequest) ([]*github.CommitFile, error)
}
// Symptom: swapping forge rewrites every caller. The import proves the leak.
```

```go
// RIGHT — the domain declares what it needs, in its own vocabulary.
type ChangeSource interface {
    FilesChanged(ctx context.Context, pr PullRequest) ([]ChangedFile, error)
}
```

Mark every block `// WRONG` or `// RIGHT` on its first line. Keep both minimal — enough to
show the mechanism, never a working program.

## The header line

Every entry opens with a `#` title and one italic line carrying **category** and
**provenance**. Provenance is not decoration — it states how much authority the entry has:

| Provenance | Means |
|---|---|
| `Derived from the global <name> skill` | Restates a standing standard. **Binding on this project.** |
| `Authored for this project` | Written from general knowledge, verified against cited sources where claims are contestable. Advisory. |
| `External: <source>` | Summarises a specification or vendor document. Authority is the source's. |

## Rules

- **No entry explains why patterns matter, what object orientation is, or what architecture
  is.** The reader knows. Describe the thing, not the field.
- **Where a claim is contested, say so and by whom.** "Widely considered an anti-pattern" is a
  fact about opinion; "is an anti-pattern" is a claim this knowledge base cannot support.
- **One entry, one subject.** A file needing "and" in its title is two files.
- **Keep an entry under roughly 110 lines.** Past that, it is a directory.
