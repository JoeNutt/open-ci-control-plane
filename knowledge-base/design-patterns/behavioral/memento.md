# Memento

*Behavioral · GoF · Authored for this project*

## What it is

An object's internal state is captured into a separate opaque object so it can be restored
later, without exposing that state to anyone else. The originator creates the memento and is
the only thing that can read it; a caretaker holds it and hands it back, treating it as a
token it cannot inspect.

## When to use it

- **Undo or rollback** is needed, and the state to restore is internal.
- Snapshots or restore points are required — a checkpoint before a risky operation, a saved
  draft, a transaction boundary.
- The state must not be exposed. The pattern's distinguishing feature versus "just serialise
  it" is that encapsulation survives.

## How to use it

Make the memento opaque to everyone but the originator, and capture a **deep enough** copy
that later mutation of the original cannot reach into it.

Go encapsulates per package, not per type: a snapshot with unexported fields is opaque to
every other package and readable by everything in the originator's own. Keep the caretaker in
a different package, or the opacity is a convention.

```go
// RIGHT — unexported fields make the snapshot an opaque token outside this package.
type Snapshot struct {
    blocks []Block
}

func (d *Document) Save() Snapshot {
    return Snapshot{blocks: slices.Clone(d.blocks)}
}

func (d *Document) Restore(s Snapshot) {
    d.blocks = slices.Clone(s.blocks)
}
```

The invariant: **the memento shares no mutable structure with the originator.** A shallow
capture means editing the document also edits its own undo history, and the symptom is undo
restoring the state you are trying to leave.

## How not to use it

- **Shallow copies.** The single most common defect, and the one that looks correct in
  testing until a nested object is edited. See `../creational/prototype.md` for the same
  hazard.

```go
// WRONG — the snapshot shares the document's backing array.
func (d *Document) Save() Snapshot { return Snapshot{blocks: d.blocks} }
// Symptom: editing a block in place edits the snapshot too, and undo restores the edit.
```

- **A caretaker that reads the memento.** Once the holder understands the contents, the state
  is public and the pattern has bought nothing over exposing fields.

```go
// WRONG — the snapshot exports its contents, so the caretaker inspects them.
type Snapshot struct{ Blocks []Block }

func (h *History) IsDocumentEmpty() bool { return len(h.latest().Blocks) == 0 }
// Symptom: History depends on Document's layout, and changing that layout breaks undo.
```

- **Unbounded history.** Snapshots of large state, kept per edit, with nothing discarding
  them, is a memory leak attached to a feature users like. Bound the depth.

```go
// WRONG — a snapshot per edit, and nothing ever discards one.
func (h *History) Push(s Snapshot) { h.snapshots = append(h.snapshots, s) }
// Symptom: memory grows with every edit in a long session until the process is killed.
```

- **Snapshotting external resources.** Open handles, sockets and live connections cannot be
  captured and restored. Capture the information needed to re-establish them instead.

```go
// WRONG — the snapshot captures a live connection.
func (s *Session) Save() Snapshot { return Snapshot{conn: s.conn, cursor: s.cursor} }
// Symptom: restoring after a reconnect revives a closed connection, and the next write fails.
```

- **Using it to persist state across processes.** Then it is a serialisation format and needs
  versioning, compatibility and validation like any other wire format. The pattern assumes
  the originator on both sides is the same code.

```go
// WRONG — the snapshot exported and written out for another process to restore.
type Snapshot struct {
    Blocks []Block `json:"blocks"`
}

encoded, err := json.Marshal(doc.Save())
// Symptom: after a release renames a Block field, old drafts restore with that field silently empty.
```

**Trade-off:** memory proportional to history depth times state size, and a capture that must
be updated whenever a field is added — a new field omitted from `save` is a silent
restore-to-wrong-state bug.

**Related:** `command.md`, `../creational/prototype.md`
