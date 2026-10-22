# Creational patterns

Who decides which concrete type gets instantiated, and when.

| Pattern | One line | Reach for it when |
|---|---|---|
| `factory-method.md` | A subclass decides which concrete product to create | A base class must create collaborators it cannot name |
| `abstract-factory.md` | One interface creates a whole family of related products | Products must be used together and mixing families is a bug |
| `builder.md` | Construction is split into steps, separate from representation | An object has many optional parts, or construction order matters |
| `prototype.md` | New objects are made by cloning an existing instance | Instantiating from scratch is expensive or the exact class is unknown |
| `singleton.md` | One instance, globally reachable | **Almost never. Read the entry before using it.** |
| `object-pool.md` | Expensive instances are reused rather than recreated | Creation cost dominates and instances are interchangeable *(non-GoF)* |
| `multiton.md` | One instance per key, rather than one overall | A fixed, keyed set of long-lived instances *(non-GoF)* |
