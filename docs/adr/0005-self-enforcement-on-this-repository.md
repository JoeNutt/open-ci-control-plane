# ADR-0005 — This repository is governed, with the exclusion kept as the deadlock escape

**Status:** Accepted · **Date:** 2026-10-14 · **Diverges from:** FR-21 · **Reconciles:** §18's
phase 1 exit criterion · **Bears on:** R-1, R-2, FR-22

## Context

The baseline requires self-enforcement in one place and forbids it in another.

> **§18, phase 1 exit criterion** — "The four MVP items working end to end **on this project's own
> repository**: a pull request refused on a diff rule with no environment provisioned, and a
> passing one running a constrained matrix dispatched by event."

> **FR-21** — "The control plane excludes its own repository and the policy repository from
> enforcement, from configuration read at start-up that no governed repository can set."

Phase 1 cannot be exited without gating this repository, and FR-21 forbids gating it. One of the
two has to give.

FR-21 is not a self-exemption control, which is what it looks like next to FR-20. It mitigates
**R-1**:

> "Admission deadlock. If the control plane governs its own repository or the policy repository, a
> policy change that breaks evaluation cannot be fixed by a pull request, because the pull request
> cannot pass."

Its stated partner is FR-22's break-glass path, and both are phase 2 work — "before any policy is
promoted to deny". So §18 asks for a refusal on this repository in phase 1, one phase before the
escape from a bad refusal exists.

## Decision

**This repository is governed.** The exclusion in FR-21 is kept, but as a switch rather than a
constant: it defaults to excluding the policy repository only, and excluding this repository is an
operator action rather than the standing state.

The policy repository stays excluded unconditionally. A policy change gating itself is R-1 with no
escape at all — the fix and the thing blocking the fix are the same commit.

Three conditions make this sound, and all three already hold:

1. **Policy is not in the governed repository.** ADR-0004 puts constraints in the policy
   repository, so a pull request here cannot weaken the gate that judges it. R-1's deadlock is
   still possible; self-exemption is not.
2. **The deciding instance is not the change under judgement.** A pull request here is judged by
   the deployed instance running released code against the policy as it already stands. The
   weakening case judges itself only if a deployment hot-loads from the branch, which nothing does.
3. **The deadlock escape does not need a pull request.** Under the self-hosted model (D-20) the
   operator has deployment access, so recovery is to flip the exclusion in start-up configuration
   and restart. That is R-2's "documented operator override", and it is available in phase 1
   without waiting for FR-22.

## Consequences

Phase 1 can be exited as §18 specifies, and the product is used on its own source, which is the
only way its refusals get exercised against real changes rather than fixtures.

**R-1 is accepted rather than removed, and the escape is weaker than FR-22's.** A bad policy will
block every pull request here until an operator changes configuration and restarts. That is
minutes of downtime for a single-operator project and would be an incident for anyone else, which
is why the exclusion remains available and why FR-22 is still phase 2 work rather than cancelled.

**This reasoning does not transfer to a hosted deployment.** Condition 3 rests on the operator
having direct access to the process. A multi-tenant deployment governing its own source repository
has no such escape between releases, so FR-21's exclusion stands there and this record does not
license turning it off.

The requirement text is unchanged and unchangeable — this records a divergence against it, which is
what `docs/adr` is for. Anyone reading FR-21 for the standing behaviour must read this with it.
