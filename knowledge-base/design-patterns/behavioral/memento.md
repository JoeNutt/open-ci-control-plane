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

```java
final class Document {
    private List<Block> blocks;

    static final class Snapshot {                  // opaque token
        private final List<Block> blocks;          // private to the originator
        private Snapshot(List<Block> b) { this.blocks = List.copyOf(b); }
    }

    Snapshot save()            { return new Snapshot(blocks); }
    void restore(Snapshot s)   { this.blocks = new ArrayList<>(s.blocks); }
}
```

The invariant: **the memento shares no mutable structure with the originator.** A shallow
capture means editing the document also edits its own undo history, and the symptom is undo
restoring the state you are trying to leave.

## How not to use it

- **Shallow copies.** The single most common defect, and the one that looks correct in
  testing until a nested object is edited. See `../creational/prototype.md` for the same
  hazard.
- **A caretaker that reads the memento.** Once the holder understands the contents, the state
  is public and the pattern has bought nothing over exposing fields.
- **Unbounded history.** Snapshots of large state, kept per edit, with nothing discarding
  them, is a memory leak attached to a feature users like. Bound the depth.
- **Snapshotting external resources.** Open handles, sockets and live connections cannot be
  captured and restored. Capture the information needed to re-establish them instead.
- **Using it to persist state across processes.** Then it is a serialisation format and needs
  versioning, compatibility and validation like any other wire format. The pattern assumes
  the originator on both sides is the same code.

**Trade-off:** memory proportional to history depth times state size, and a capture that must
be updated whenever a field is added — a new field omitted from `save` is a silent
restore-to-wrong-state bug.

**Related:** `command.md`, `../creational/prototype.md`
