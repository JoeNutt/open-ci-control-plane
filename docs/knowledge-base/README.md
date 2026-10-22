# Knowledge base

Reference material for the Open CI Control Plane, organised so a session reads one entry at a
time. **Nothing here is imported into context automatically** — it is read on demand, which is
the whole reason it is split this finely.

Every entry follows `FORMAT.md`. Read that before adding one.

| Area | Holds | Start at |
|---|---|---|
| `architecture/` | Dependency direction, ports and adapters, coupling, cohesion, boundaries, testability, deployability, proportionality, incremental change. Derived from the global `architecture` skill. | `architecture/README.md` |
| `engineering/` | How code gets written: red-green-refactor, naming, routines, control flow, classes, scope, comments, layout, duplication, table-driven methods, the barricade, errors and assertions. Derived from the global `tdd` and `cc2-*` skills. | `engineering/README.md` |
| `security/` | Secure coding: threat modeling, input validation, injection, authorization, least privilege, cryptography, secrets, error handling and logging, supply chain, review and testing. Derived from the global `sec-*` skills, plus the **OWASP Top 10:2025** as its own subdirectory. | `security/README.md` |
| `design-patterns/` | The Gang of Four catalogue plus a marked set of later additions. One file per pattern. | `design-patterns/README.md` |
| `technology/` | The engine this project depends on: OPA and Rego, pinned to v1.21.2. | `technology/README.md` |
| `process/` | How work is taken and tracked: reconciling the project board each session. The owner's direction, binding. | `process/README.md` |

## Provenance decides authority

Entries carry different weight, and the header line of each one says which:

- **Derived from a global skill** — restates a standing standard. **Binding.** The skill
  remains the authority; if an entry disagrees with its skill, the skill wins and the entry is
  the bug.
- **Authored for this project** — written from general knowledge, verified against cited
  sources where claims are contestable. **Advisory.**
- **External** — summarises a specification. Authority is the source's.

Do not treat an advisory entry as a standard, and do not weaken a standard because an advisory
entry disagrees with it.

## Known debt

One set of entries predates the current `FORMAT.md` and does not comply. (The 26 design-pattern entries were brought into line in #108 and #109.) This is recorded rather
than hidden, because an entry that looks compliant and is not is worse than one known to be
stale.

| Entries | What is wrong | Fix |
|---|---|---|
| `technology/opa-embedding-and-prior-art.md`, `technology/rego-language.md` | Organised by subject rather than the five sections. | Left deliberately. These are technology dossiers, not single-subject entries; splitting them would scatter findings that only make sense together. Their two overview files are the entry points. |
