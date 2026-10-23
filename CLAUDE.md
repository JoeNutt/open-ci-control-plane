# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
| The specification | the artifacts linked below | on demand, by reading it |
| Reference material | `docs/knowledge-base/` — see its `README.md` and `FORMAT.md` | on demand, by reading it |
| Engineering standards | `~/.claude/skills/` | per the routing table in the global `CLAUDE.md` |

Paths above are in backticks deliberately. An unbackticked `@path` is an **import**, expanded
into context at launch — the opposite of routing. Keep imports out of this file.

## Work order

**At the start of every session, reconcile the board first** — every card in the column its
state earns, blocked cards with their reason — as `docs/knowledge-base/process/board-reconciliation.md`
describes. Then take the **Ready card with the lowest `Sequence`** on the board
(https://github.com/users/JoeNutt/projects/2) and work only that card. The sequence is the
dependency order: card N needs only cards before it. Cards without a `Sequence` are not work
yet. A card closes only when every obligation in its checklist is met against the SAD — the
full rule is in `.claude/rules/tdd-cycle.md`, "One card at a time".

## Read before you change

| Changing | Read first |
|---|---|
| A component, boundary or dependency | the specification, §12 design and §20 decision record |
| Anything the specification decides | The current release's §20 decision record. The baseline spec is frozen; a release's §20 says where it overrides it |
| Which card to pick up, or what blocks it | The current release's §18 build order, the board's `Sequence`, and the one-card rule in `.claude/rules/tdd-cycle.md` |
| An HTTP contract | §13, and `D-21` before adding any endpoint |
| The policy input document | §13 `C-2`. It is a published cross-release contract — additive only |
| Anything touching Rego or OPA | `docs/knowledge-base/technology/rego-overview.md`, then `technology/opa-overview.md` |
| Where code belongs, or crossing a layer | `docs/knowledge-base/architecture/README.md`, then the one entry |
| Writing or changing any testable code | `docs/knowledge-base/engineering/tdd-red-green-refactor.md`. Refactor is a step, not an afterthought |
| Any code at all — naming, routines, comments, duplication | `docs/knowledge-base/engineering/README.md`, then the one entry |
| Untrusted input, authorisation, secrets, a query or a command | `docs/knowledge-base/security/README.md`, then the one entry |
| Checking a finished design for security | `docs/knowledge-base/security/owasp-top-10/README.md`, all ten |
| Applying a design pattern | `docs/knowledge-base/design-patterns/README.md`, then the one entry |
| Adding a knowledge base entry | `docs/knowledge-base/FORMAT.md`. Five sections, Go examples only, wrong-code mandatory |
| A bound, limit or timeout | §8. Bounds are numbered requirements, not implementation detail |
| Test selection | §9's always-run set and confidence triggers before changing what gets skipped, then `docs/knowledge-base/test-selection/README.md` |

Baseline specification, frozen: https://claude.ai/code/artifact/db03f5d3-b4e0-4fb0-8254-375c6cf0d08d.

**Current release: v0.0.2**, GitHub milestone 9, specified at
https://claude.ai/artifact/JoT3JYoqXSLExzsLERjrSN. Its requirements are the milestone's
`requirement` issues, and where a spec and an issue disagree, the issue wins. v0.0.1 shipped,
specified at https://claude.ai/artifact/CjiuxFHroNTwjRNhpnpQ7U.

The release spec narrows the baseline, and its §20 decisions override it where they say so.
Its §18 is the build order. Read it before taking a card.

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
| `.claude/hooks/refactor-verdict.sh` | `Stop`, `SubagentStop` | Ending a turn that edited source without stating a refactor verdict. `mark-source-edit.sh` (`PostToolUse`) records which non-test `.go`/`.rego` files were edited |
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

Module `github.com/JoeNutt/open-ci-control-plane`, Go 1.26 — the floor OPA v1.21.2 declares.
**`go` is not on the bare PATH**: it comes from the flake, so every Go command runs in the
dev shell. Batch them, because the shell is slow to start.

```
nix develop --command bash -c 'gofmt -l $(git ls-files "*.go") && go vet ./... && go test ./...'
nix develop --command bash -c 'golangci-lint run ./...'
nix develop --command go test ./internal/change/domain -run 'TestName' -v
```

The commit gate runs `gofmt`, `go vet` and `go test` itself, using `go` from PATH if present
and the flake otherwise. Git does not enable it on clone: run `git config core.hooksPath
.githooks` once per checkout. The suite shells out to the `opa` CLI (the schema conformance
test) and `cmark-gfm` (the check-run renderer test), which is another reason it only passes
in the dev shell. The shell sets `GOFLAGS=-mod=readonly` and `CGO_ENABLED=0`.

The binary is `cmd/controlplane`. It takes exactly one argument, the path to a JSON
configuration file (`internal/change/configuration`), logs JSON to stderr, and serves
`POST /v1/forge/github/events` and `PUT /v1/repositories/{owner}/{repo}/coverage/{suite...}`.
`openapi.yaml` declares more paths than the binary serves. A webhook is answered at once, and
the evaluation runs in the background under a bounded timeout (`cmd/controlplane/evaluation.go`).

## Code layout

`cmd/controlplane` is the composition root. Everything else lives under `internal/change/`,
arranged as ports and adapters:

- `domain/` owns the types and the ports, each in its own file. Its constructors are
  `Parse*`, which return domain values. `architecture_test.go` fails the suite if a domain
  package imports anything except the standard library and other `domain` packages.
- `application/` holds the use cases. `BuildDocument` assembles the policy input document
  from the change, the structural facts extracted from the changed source, and the
  classification rules. `GateChange` evaluates the document per policy family, and for each
  decision both records it and publishes one check run. It does this under
  `context.WithoutCancel` and joins the failures, so one family's failure never stops
  another's record or check run. On a structural pass it plans the test matrix with
  `PlanTests` and dispatches the plan. A planning error dispatches the full matrix
  (`domain.FailedPlanFor`), never a selection, and records the dispatched plan in the plan
  ledger. `RecordRun` takes a run's report, refuses one for a head with no recorded plan or
  one already reported, and concludes the `open-ci/tests` check run. `ReplaceCoverage` authorises a coverage upload against the
  caller's identity token, applies the `UploadRate` limit, and replaces one suite's coverage.
- Inbound adapters: `httpapi` (routes, problem responses, replay protection, installation
  allowlist), `ingest` (webhook signature check, GitHub Actions identity tokens) and
  `configuration` (the bounded JSON configuration file).
- Outbound adapters: `forge/github` (pull requests, check runs, plan dispatches, App JWT and
  installation tokens), `history/jsonl` (the bounded shadow history log, and the plan ledger
  kept beside it), `engine/opa` (one embedded OPA instance per policy family; `engine/opa/structural` embeds the structural Rego rules), `policy`
  (the central catalog and per-repository bindings, validated against
  `schema/policy.*.v1.schema.json`), `coverage/lcov` and `coverage/memory`, `extract/golang`
  (Go imports), `jsonlines` (decision records), `rules` (classification rules) and `random`
  (decision IDs).

Every `.go` file starts with `// SPDX-License-Identifier: GPL-3.0-or-later`.

`openapi.yaml` is the HTTP surface. `schema/ci.diff.v1.schema.json` is the policy input
contract, and emitted documents are tested against it.
