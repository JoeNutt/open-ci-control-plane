# Delivery phases

The board's hierarchy is the specification's, not one invented here. §18 *Delivery approach*
defines six phases and names the functional requirements each delivers. Every card carries a
milestone matching its phase, so progress reads per phase rather than per card number.

**A card's milestone is the latest phase naming any of its requirements** — the phase in which
it can close. Two cards span phases and are noted below.

## Phases, as §18 states them

| Milestone | Delivers | Requirements |
|---|---|---|
| Phase 1 — The MVP artifact | A thin vertical slice of all three pillars, not one pillar whole | FR-1–FR-12, FR-17, FR-18, FR-29–FR-31, FR-34, FR-35, FR-37, FR-38, FR-49, FR-51, FR-65, FR-74, FR-76, FR-78–FR-80, FR-83, FR-84, FR-88 |
| Phase 2 — Enforcement and recovery | Warn then deny, three-way split, self-exemption guard, break-glass, shadow mode, telemetry | FR-13–FR-16, FR-19–FR-22, FR-36, FR-90–FR-95 |
| Phase 3 — The structural family | Arrives whole, because its parts are useless separately | FR-47–FR-73 |
| Phase 4 — Distribution, scale and supply chain | Bundle integrity, signatures, audit runs, signed releases, SBOM | FR-23–FR-28, FR-32, FR-33, FR-101–FR-105 |
| Phase 5 — Selection enabled, and breadth | Pillar B for a third party, further extractors, tier 2, inline comments | FR-71, FR-72, FR-75, FR-77, FR-81, FR-82, FR-85–FR-87, FR-95 |

Phase 0 — Baselines has no card: it is Q-1, Q-6 and Q-8 research, runs in parallel, and §18
states it is not a gate on phases 1 or 2.

Ordering is dependency, not importance. §18 says so of phase 3 explicitly: the structural family
is the headline and is late because the rules need the extractors, the model contract and the
template mechanism beneath them.

## Blocking, within a phase

A milestone says *when*; these say *what first*. Each edge is justified by requirement text, not
by preference.

| Card | Blocked by | Because |
|---|---|---|
| #2 FR-3, FR-5, FR-9 | #5 | FR-3 requires policy be evaluated against the document and FR-9 that a major mismatch be reported *as a policy error*. Both need FR-10's engine. FR-5 is already met. |
| #8 FR-17, FR-18 | #5 | FR-17 posts "the verdict, each rule that fired, its owning team, its documentation link and the decision identifier" — every field is produced by evaluation. |
| #13 FR-29–FR-31 | #5 | FR-29 records "result, the identifiers of the rules that fired, and a decision identifier". |
| #3 FR-4 | #8 | FR-4 refuses an oversized change "posting a check run naming the bound that was breached". |
| #56 FR-84 | #5, #8 | FR-84 posts an in-progress check run "and updates it with the verdict when evaluation completes". |
| #31 FR-83, FR-88 | #29 | Dispatching a constrained matrix needs the test plan FR-34 produces. |
| #28 FR-74, FR-76 | #5, #31 | FR-76 — "dispatches no test matrix for a change the gate refused" — constrains dispatch, so it needs both a gate verdict and dispatch to exist. |

**Phase 1 cards with nothing blocking them:** #1, #4, #5, #29, #30, #37, #38.

## Two gaps in the baseline, not papered over

**§18 sequences functional requirements only.** No NFR is assigned to a phase anywhere in the
specification. Twenty NFR cards therefore sit in *Unphased in the baseline* rather than being
guessed into a phase. Several have an obvious coupling — NFR-5's bounds are what FR-4 enforces,
NFR-13's TLS is needed the moment anything serves — and each wants a deliberate decision rather
than an inference.

**FR-46 (#62) and FR-89 (#57) are named by no phase.** Both are in §9's requirements table and
neither appears in any phase's list. Same holding milestone, same reason.

## Won't Have is recorded in the specification, not on the board

The twelve Won't Have requirements are closed as *not planned*. §9's table carries each with its
rationale, §18 excludes the twelve from the estimate base, and the "Won't Have, this time"
section states why each is out. A second copy on the board was duplication with no owner, and
twelve cards that will never move made progress harder to read. Bringing one into scope means
reopening it, which is the deliberate signal that was wanted.
