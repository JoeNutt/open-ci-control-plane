# Design patterns

One file per pattern, so a session reads the one it needs. Every entry follows
`../FORMAT.md`: what it is, when to use it, how to use it, how not to use it.

| Category | Composes | Entries |
|---|---|---|
| `creational/` | How objects get made, and who decides which concrete type | 5 GoF + 2 later |
| `structural/` | How objects and classes are assembled into larger structures | 7 GoF |
| `behavioral/` | How responsibility and control flow are distributed between objects | 11 GoF + 1 later |

## What is in the catalogue, and what is not

The **23 Gang of Four patterns** are the canonical set — 5 creational, 7 structural, 11
behavioral — and the membership is not in dispute across sources
([GoF catalogue](https://www.geeksforgeeks.org/system-design/gang-of-four-gof-design-patterns/)).

A few later patterns are included and **marked as non-GoF in their header line**, because
they are widely used and a reader looking for them should find them rather than conclude
they do not exist. Their categorisation is weaker than GoF membership:

| Pattern | Category used here | Standing |
|---|---|---|
| Object Pool | creational | Consistently classified creational across catalogues ([Wikipedia](https://en.wikipedia.org/wiki/Object_pool_pattern)) |
| Multiton | creational | Generalises Singleton, so grouped with it; no source states the category outright ([Wikipedia](https://en.wikipedia.org/wiki/Multiton_pattern)) |
| Null Object | behavioral | Catalogues split between behavioral and structural. Either is defensible |

Deliberately excluded: **Simple Factory**, a common idiom frequently miscounted as GoF but
not one of the 23; and **Dependency Injection**, which is an architectural technique with no
agreed GoF category and is better treated under architecture than here.

## On "obsolete" patterns

Several entries note that a pattern has been absorbed into language features — lambdas
largely removing Strategy's boilerplate, prototype chains subsuming Prototype, container
lifetimes replacing Singleton. **Treat those notes as expert opinion, not settled
consensus.** The sources making the strongest obsolescence claims are blog posts and forum
threads, and the counter-argument — that patterns became infrastructure developers consume
rather than implement, and that over-engineering is now the larger risk — is argued just as
forcefully. Where an entry's `How not to use it` section rests on this, it says so.
