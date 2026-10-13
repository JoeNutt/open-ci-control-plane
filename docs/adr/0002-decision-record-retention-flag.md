# ADR-0002 — Add an explicit retention flag to the decision record

**Status:** Accepted · **Date:** 2026-10-13 · **Diverges from:** specification §13, C-8

## Context

C-8 requires that `maskedInput` is omitted once the diff retention window has passed, and
that "the response says so rather than returning an empty object". It states the requirement
and names no field to carry it.

Without one, a consumer cannot distinguish three cases that must not be confused: input was
never stored, input was stored and has been dropped for retention, and input is present but
every pointer in it was redacted. The third is already covered by `redactedPointers`; the
first two are not.

## Decision

The decision record carries `inputRetained` (boolean): false when `maskedInput` was dropped
for retention rather than never stored.

## Consequences

This is an **addition** to the baseline, not a transcription of it. It was introduced in
`openapi.yaml` and is recorded here because an invented field appearing first in a generated
type is exactly how a contract acquires surface nobody decided on.

If a reviewer prefers a different realisation — an explicit `inputState` enum over a boolean,
say — this record is the thing to change. It is cheap now and expensive once a tenant depends
on the field.
