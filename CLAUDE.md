# Open CI Control Plane

A self-hosted Go service that sits between a git forge and a CI runner. Three capabilities:
a **deterministic policy gate** (structural, security and quality policy families), **test
matrix optimisation** from coverage artifacts, and **orchestration and telemetry** for an
external developer portal.

Policy engine is OPA v1.21.2, embedded via `opa/v1/sdk`. Policy is Rego, authored mostly as
parameters against templates rather than written directly.

## This file is a router

It carries only what every session needs. Everything else is read on demand. **Resist adding
detail here** — put it where it belongs and route to it.

| Kind of knowledge | Lives in | Loaded |
|---|---|---|
| Always-true project facts, this table | `CLAUDE.md` | every session |
| Rules for one area of the tree | `.claude/rules/*.md` with `paths:` frontmatter | only when a matching file is touched |
| Enforcement that must not be skippable | `.claude/hooks/` wired in `.claude/settings.json` | at the lifecycle event, regardless of what I decide |
| The specification | `docs/` | on demand, by reading it |
| Reference material | `knowledge-base/` — see its `README.md` and `FORMAT.md` | on demand, by reading it |
| Engineering standards | `~/.claude/skills/` | per the routing table in the global `CLAUDE.md` |

Paths above are in backticks deliberately. An unbackticked `@path` is an **import**, expanded
into context at launch — the opposite of routing. Keep imports out of this file.

## Read before you change

| Changing | Read first |
|---|---|
| A component, boundary or dependency | the specification, §12 design and §20 decision record |
| An HTTP contract | §13, and `D-21` before adding any endpoint |
| The policy input document | §13 `C-2`. It is a published cross-release contract — additive only |
| Anything touching Rego or OPA | `knowledge-base/technology/rego-overview.md`, then `technology/opa-overview.md` |
| Where code belongs, or crossing a layer | `knowledge-base/architecture/README.md`, then the one entry |
| Writing or changing any testable code | `knowledge-base/engineering/tdd-red-green-refactor.md`. Refactor is a step, not an afterthought |
| Any code at all — naming, routines, comments, duplication | `knowledge-base/engineering/README.md`, then the one entry |
| Untrusted input, authorisation, secrets, a query or a command | `knowledge-base/security/README.md`, then the one entry |
| Checking a finished design for security | `knowledge-base/security/owasp-top-10/README.md`, all ten |
| Applying a design pattern | `knowledge-base/design-patterns/README.md`, then the one entry |
| Adding a knowledge base entry | `knowledge-base/FORMAT.md`. Five sections, Go examples only, wrong-code mandatory |
| A bound, limit or timeout | §8. Bounds are numbered requirements, not implementation detail |
| Test selection | §9's always-run set and confidence triggers before changing what gets skipped |

Specification: https://claude.ai/code/artifact/db03f5d3-b4e0-4fb0-8254-375c6cf0d08d —
not yet exported to `docs/`.

## Invariants

Load-bearing, non-obvious, and not derivable from the code. Everything else is in the global
standards.

- **Undefined is not false.** An empty OPA result set means the policy did not decide. It is
  never an admit. Branch on it explicitly, everywhere.
- **Unreachable fails open; evaluation errors fail closed.** The unreachable branch is not
  configurable. Per-policy fail-closed governs decision *outcomes* only.
- **Reject, never truncate.** A truncated diff is one no policy ever saw.
- **`opa/v1/...` import paths only.** `opa/rego` is deprecated and defaults to v0 semantics.
- **No architectural style is built in.** "Layer", "port" and "adapter" appear only in a
  tenant's own data document and in opt-in example packs, never in the engine.
- **Policy lives outside the repository it governs.** Nothing read for a decision may come
  from the repository being decided about.
- **Never commit in a working tree an agent is live in.** Spawn agents with
  `isolation: "worktree"` so they get their own checkout. Two writers in one tree is a
  concurrency bug: the branch moves under you, and a stray file in the repo root breaks
  `go test ./...` module-wide for whoever else is working there. Both have already happened.

## The cycle is enforced, not suggested

`.claude/rules/tdd-cycle.md` loads whenever a `.go` or `.rego` file is touched and carries the
three steps in full. Two hooks make it mechanical rather than advisory:

| Hook | Event | Refuses |
|---|---|---|
| `.claude/hooks/refactor-verdict.sh` | `Stop`, `SubagentStop` | Ending a turn that edited source without stating a refactor verdict |
| `.githooks/pre-commit` | `git commit` | Unformatted files; `go vet` failures; a red suite |

The second is an ordinary git hook, wired by `core.hooksPath`, so it binds humans as well as
agents. Only the first has to be a Claude hook — it reads assistant output, which nothing else
can see.

The verdict wording is checked mechanically. Exactly one of:

```
Refactored: <what changed and why>
No refactor needed: checked <which questions>, none fired.
```

## Commands

No Go module yet. Add build, test and lint commands here when they exist. Until `go.mod`
exists the commit gate skips the Go checks and still enforces the test-alongside-code rule.
