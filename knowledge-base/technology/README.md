# Technology reference

The policy engine this project embeds. All API detail pinned to **OPA v1.21.2** (released
2026-10-12), produced by executing the binary and reading the tagged source tree rather than
transcribing documentation. Each entry marks what it could not verify.

| Entry | Read it for |
|---|---|
| `opa-overview.md` | What OPA is, the vocabulary, deployment modes, why this project uses it. **Start here.** |
| `opa-embedding-and-prior-art.md` | The Go embedding API, performance and memory, bundles and signing, the missing resource limits, error modes, prior art, engine alternatives, project health. |
| `rego-overview.md` | What Rego is, where a policy lives, the smallest complete example. **Start here for policy.** |
| `rego-language.md` | The undefined hazard in full, version gotchas, input document design, performance cliffs, classify-then-decide, testing, guardrails, limitations. |

The two long entries are organised by subject rather than `FORMAT.md`'s five sections. That is
deliberate and recorded as such in `../README.md` — they are a technology dossier rather than
a single-subject entry.
