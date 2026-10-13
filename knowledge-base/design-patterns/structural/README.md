# Structural patterns

How objects and classes are assembled into larger structures, and where the seams go.

| Pattern | One line | Reach for it when |
|---|---|---|
| `adapter.md` | Translates one interface into another | An existing type is right but its interface is wrong |
| `bridge.md` | Splits an abstraction from its implementation so both vary | Two dimensions of variation are multiplying subclasses |
| `composite.md` | A tree where a group and a leaf share one interface | Clients should not care whether they hold one thing or many |
| `decorator.md` | Wraps an object to add behaviour, preserving its interface | Behaviour must compose at runtime, per instance |
| `facade.md` | One simple interface in front of a complicated subsystem | A common path through a subsystem needs naming |
| `flyweight.md` | Shares immutable intrinsic state across many instances | Object count, not object cost, is the problem |
| `proxy.md` | Stands in for another object and controls access to it | Access needs mediating: laziness, remoteness, permission |

**Adapter is the one this project uses by name.** The ports-and-adapters arrangement in the
architecture is the same translation idea applied at a module boundary rather than at a class.
