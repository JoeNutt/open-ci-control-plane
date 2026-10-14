# ADR-0004 — A policy repository, reconciled, for constraint authoring

**Status:** Accepted · **Date:** 2026-10-14 · **Serves:** FR-20, FR-21, NFR-25, §13 contract C-11
· **Diverges from:** nothing — records a decision the baseline leaves open

## Context

C-11 defines how a constraint reaches the control plane: `PUT /v1/constraints/{constraintId}`
with an operator token holding `policy:bind`, carrying a `templateId`, a `selector`, `parameters`
and `actions`. It defines no way for a human to author one. FR-70 ships example packs and tier 1
— where §4 says most adopters are expected to stay — is "enables a shipped pack and supplies
parameters", but nothing turns that into a file someone edits and reviews. FR-40, a guided web
policy editor, is Won't Have.

So the authoring experience today is hand-written JSON against an HTTP API, which nobody will do
twice.

Two tempting answers were rejected.

**A `.openci/` directory inside the governed repository.** This is the defect the product exists
to remove. Every control it replaces lives inside the repository it protects, so the agent whose
output it constrains can edit it, and "make CI pass" is a reasonable-looking instruction that a
weakened rule satisfies. FR-20 refuses any request that would weaken enforcement for the
repository it concerns without the operator credential, and C-11 states that a token issued to a
governed repository can never hold `policy:bind`. A policy directory the repository's own CI can
write would route around both.

**Generating Rego from YAML.** C-11's template already *is* Rego — `ruleBody` — and parameters
are data it reads. Generating rule source from a document instead makes the generator a trusted
compiler: a defect in it silently weakens every constraint derived from it, and no parameter
schema can catch that, because the logic rather than the input would be wrong. Parameters
validated against a template's `parameterSchema` cannot change what a rule does. Generated source
can. It also loses the property §9 relies on, that policy compilation is schema-checked against a
fixed rule and a published input schema, and it would move FR-15's capability allowlist from once
per template to once per generated constraint.

## Decision

Constraints are authored as YAML in a **policy repository**, separate from every governed
repository, and reconciled into the control plane by a pipeline in that repository.

FR-21 already names this repository — it "excludes its own repository and **the policy
repository** from enforcement" — so this specifies how an artefact the baseline already assumes
is used, rather than introducing one.

1. An operator edits YAML files in the policy repository. Each file is a constraint instance:
   `templateId`, `selector`, `parameters`, `actions`.
2. Agents working in governed repositories have no access to it. The guarantee is the credential,
   not the directory: nothing holding only a governed repository's CI token can write a
   constraint.
3. The pipeline converts each file to JSON and calls `PUT /v1/constraints/{constraintId}` with an
   operator token holding `policy:bind`. `constraintId` is the file's own identifier, so the write
   is idempotent and a re-run changes nothing.
4. The YAML supplies **parameters and selectors only**. No Rego is generated, and the templates it
   names are the pre-compiled ones in the signed bundle.

The reconciler is a **client of the published API and nothing else** — no server-side code of this
project's. That is the shape NFR-25's acceptance test already requires of a frontend, so the
authoring layer is replaceable and the API stays the single contract.

## Consequences

The security property holds for the reason the product exists: an agent editing files in a
governed repository cannot reach the rules that judge its output, because they are not in that
repository and the token that could change them is not in its CI.

Policy changes get review, history and attribution for free, which is what C-11's audit
expectations assume and what a hand-rolled JSON call does not provide.

A repository, rather than any other store a governed repository cannot write, because review and
history are what make a policy change safe to make. One adopter, one policy repository.

Where one person is both operator and author they can always change their own policy, which A-1's
assumption about teams inside an organisation already implies. What survives in that case is the
defect this project set out to fix: an *agent* cannot reach the policy, even though the human can.

**No tool is built for this.** The reconciler is a workflow in the policy repository — convert each
file with `yq` and call the endpoint with an operator token from that repository's secrets. The
YAML shape is C-11's request body in a different serialisation, so there is nothing to design and
nothing to maintain beyond the workflow file.
