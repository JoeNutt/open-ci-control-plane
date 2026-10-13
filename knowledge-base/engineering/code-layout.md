# Code layout

*Construction · Derived from the global `cc2-construction` skill — **binding on this project***

## What it is

**Read a file top-down like a newspaper:** the most important, most abstract material at the
top, detail descending beneath it. **Follow the stepdown rule** — each function is followed by
those one level of abstraction below it, so the file reads as a continuous descent. **Hold one
level of abstraction per routine.** Separate logical paragraphs with blank lines; keep
vertical distance short between things used together.

## Why it exists

The one-level-of-abstraction rule has a specific mechanism worth knowing, because it explains
why this matters more than it looks. Every time a reader drops from a high-level line to a
low-level one, they must push their current train of thought aside to deal with the detail,
then pop back to it. **People do not have a mental stack.** What gets pushed aside is usually
lost, and the reader starts the paragraph again.

A routine that alternates levels makes them do this repeatedly — an abstraction roller
coaster. The newspaper structure exists so a reader can stop after the first screen with a
correct idea of what the file does.

## When to use it

Every file, and specifically when a routine both orchestrates named steps *and* does
low-level work: zeroing counters, indexing arrays, formatting strings. That mixture is the
symptom.

## How to use it

```go
// RIGHT — top level reads as a sequence of intentions. One altitude.
func (i Ingest) BuildDocument(ctx context.Context, e Event) (Document, error) {
    files, err := i.fetchChangedFiles(ctx, e)
    if err != nil {
        return Document{}, err
    }

    bounded, err := i.enforceBounds(files)
    if err != nil {
        return Document{}, err
    }

    return i.assemble(e, bounded), nil
}

// ...and the next level down follows it, in call order.
func (i Ingest) fetchChangedFiles(ctx context.Context, e Event) ([]ChangedFile, error)
func (i Ingest) enforceBounds(f []ChangedFile) (Bounded, error)
func (i Ingest) assemble(e Event, b Bounded) Document
```

**The invariant: one routine, one altitude.** A function that calls named steps should not also
manipulate indices.

## How not to use it

**Mixing abstraction levels in one routine.**

```go
// WRONG — orchestration and byte-level detail in the same body.
func (i Ingest) BuildDocument(ctx context.Context, e Event) (Document, error) {
    files, err := i.fetchChangedFiles(ctx, e)        // high level
    if err != nil { return Document{}, err }

    total := 0
    for n := 0; n < len(files); n++ {                // low level
        total += len(files[n].Patch)
        if files[n].Path[0] == '/' {
            files[n].Path = files[n].Path[1:]        // lower still
        }
    }

    return i.assemble(e, files), nil                 // high level again
}
// Symptom: the reader rides from policy to index arithmetic and back twice.
// Extract the loop, name it `normalisePaths`, and the top level reads cleanly.
```

**Helpers above their callers.**

```go
// WRONG — the file opens with the lowest-level detail.
func normalisePath(p string) string { /* ... */ }
func parsePatchBytes(b []byte) int  { /* ... */ }
func assemble(...) Document         { /* ... */ }
func BuildDocument(...)             { /* the thing the file is about, at the bottom */ }
// Symptom: a reader must scroll to the end to find out what the file does.
```

**No vertical structure.** Forty lines with no blank line means nothing signals where one
thought ends. Related lines stay tight; unrelated lines get air.

**Reformatting unrelated code in a change.** It buries the real edit in noise and makes the
diff unreviewable. Use the project's formatter and leave the rest alone — **a consistent style
you dislike beats a mix of styles you like.**

**Trade-off:** the stepdown rule sometimes conflicts with grouping related small functions
together, and Go's convention of exported-first can pull against strict call order. Idiom and
the project formatter win; the newspaper shape is the default where nothing conflicts.

**Related:** `comments.md`, `routines.md`, `classes-and-abstractions.md`
