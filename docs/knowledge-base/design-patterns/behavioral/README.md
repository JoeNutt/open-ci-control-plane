# Behavioral patterns

How responsibility and control flow are distributed between objects.

| Pattern | One line | Reach for it when |
|---|---|---|
| `chain-of-responsibility.md` | A request passes along handlers until one takes it | The handler is not known at the call site |
| `command.md` | A request becomes an object | Operations must be queued, logged, undone or retried |
| `interpreter.md` | A grammar is represented as a class per rule | You own a small language and need to evaluate its trees |
| `iterator.md` | Traversal is separated from the collection | Clients must walk a structure without knowing its shape |
| `mediator.md` | A hub owns the interaction between components | Many-to-many coupling has become a web |
| `memento.md` | State is captured externally without exposing internals | Undo, snapshots, or restore points |
| `observer.md` | Dependents are notified when a subject changes | One change must reach an unknown set of listeners |
| `state.md` | Behaviour changes with an object's state, via delegation | A conditional over a state field appears in many methods |
| `strategy.md` | An algorithm is selected at runtime behind one interface | The *how* varies and the *what* does not |
| `template-method.md` | A base class fixes the steps; subclasses fill them in | A sequence is invariant and its steps are not |
| `visitor.md` | An operation is added to a structure without changing it | Operations change often, the node types rarely |
| `null-object.md` | A neutral implementation replaces absence | Null checks are duplicated everywhere *(non-GoF)* |

Several of these have been partly absorbed by language features — see the obsolescence note
in `../README.md` before concluding a pattern is dead.
