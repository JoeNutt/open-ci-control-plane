# Board reconciliation

*Process · The project owner's direction, 2026-10-14 — **binding on this project***

## What it is

At the start of every session, before any card is taken, every open card on the project board
(https://github.com/users/JoeNutt/projects/2, a Kanban board whose columns are `Status`) is put
in the column its real state earns, and the board is put in work order. Then the Ready card
with the lowest `Sequence` is the only card worked.

## Why it exists

Because a board that drifts stops saying what can be done. Cards were picked from anywhere in
the backlog, the Issues page lists newest first and buried every Must, and card #8 was started
while it depended on cards later in every phase — 238 commits, closed, then reopened incomplete.
A future session can only pick up where the last left off if the board is the truth.

## When to use it

- At the start of every session, before taking a card.
- When a card closes: the next card's dependencies may now be Done.
- When a card is re-scoped, created, or found to depend on something no card provides.

## How to use it

Each column has one meaning. Decide it from the card's checklist and the SAD, never from mood.

| Column | A card belongs here when |
|---|---|
| `Done` | Its issue is closed on proof that every checklist obligation is met |
| `In Progress` | It is the one card being worked — at most one |
| `Ready` | Every card it depends on is Done and nothing it needs is an open gap |
| `Blocked` | It needs something no card provides — a SAD gap, an open decision, a dependency loop — or it is started and stuck. **`Blocked by` states why**, always |
| `Backlog` | It waits only on other cards |

Board order: sequenced cards by `Sequence`; then every Must, every Should, every Could, each by
phase, then card number. Work only the Ready card with the lowest `Sequence`.

```sh
# RIGHT — read the board's true state before acting. `gh project item-list` leaves text fields
# such as "Blocked by" empty, so read every field through the API (100 items per page;
# paginate past that).
gh api graphql -f query='{ user(login:"JoeNutt"){ projectV2(number:2){ items(first:100){ nodes{
  content{ ... on Issue{ number title state } }
  status: fieldValueByName(name:"Status"){ ... on ProjectV2ItemFieldSingleSelectValue{ name } }
  sequence: fieldValueByName(name:"Sequence"){ ... on ProjectV2ItemFieldNumberValue{ number } }
  blockedBy: fieldValueByName(name:"Blocked by"){ ... on ProjectV2ItemFieldTextValue{ text } }
} } } } }'
```

```sh
# RIGHT — move a card and say why it is blocked, in the same pass.
gh project item-edit --project-id "$PROJECT" --id "$ITEM" --field-id "$STATUS" --single-select-option-id "$BLOCKED"
gh project item-edit --project-id "$PROJECT" --id "$ITEM" --field-id "$BLOCKED_BY" --text "No contract defines the activation trigger NFR-10 requires"
```

```sh
# RIGHT — keep work order physical, so the board reads top-down.
gh api graphql -f query='mutation($p:ID!,$i:ID!,$a:ID!){updateProjectV2ItemPosition(input:{projectId:$p,itemId:$i,afterId:$a}){clientMutationId}}' \
  -f p="$PROJECT" -f i="$ITEM" -f a="$PREVIOUS"
```

**The invariant: the board is true before work starts, and only the top Ready card is worked.**

## How not to use it

**Taking work from the Issues page.**

```sh
# WRONG — the Issues page is newest first; it buried every Must under later Shoulds.
gh issue list --limit 20        # then picking whatever looks next
# Symptom: a Could is started while the Must it depends on sits on page three.
```

**Marking a card Ready without checking what it depends on.**

```sh
# WRONG — Ready by intuition.
gh project item-edit ... --single-select-option-id "$READY"    # card #8, 2026-10-13
# Symptom: 238 commits, then the card cannot close: its §14 rows need cards later in every phase.
```

**Blocked with no reason.**

```sh
# WRONG — a column of fifty cards nobody can act on.
gh project item-edit ... --single-select-option-id "$BLOCKED"  # Blocked by left empty
# Symptom: the next session cannot tell a SAD gap from a missing card, so it guesses.
```

**Closing a card by moving an unmet obligation elsewhere.** Scope moves happen before work, on
both cards; at closing time an unmet obligation keeps the card open — see
`.claude/rules/tdd-cycle.md`, "One card at a time".

**Trade-off:** a minute at the start of every session, and Blocked reasons that must be kept
current — cheaper than one card worked out of order.

**Related:** `../../../CLAUDE.md` "Work order", `../../../.claude/rules/tdd-cycle.md`
