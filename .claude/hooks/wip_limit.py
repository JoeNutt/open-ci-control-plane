"""Decides whether a board edit would breach the one-card-per-agent WIP limit.

Reads the board field map, a GraphQL snapshot of the project's items, and the
Bash command being attempted. Prints a PreToolUse deny decision to stdout when
the limit would be breached, and nothing when the call is allowed.
"""

import json
import re
import sys


def allow():
    sys.exit(0)


def deny(reason):
    print(
        json.dumps(
            {
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": "deny",
                    "permissionDecisionReason": reason,
                }
            }
        )
    )
    sys.exit(0)


def selected_option(item, field_id):
    for value in item.get("fieldValues", {}).get("nodes", []):
        if (value.get("field") or {}).get("id") == field_id:
            return value.get("optionId")
    return None


def main():
    board_path, snapshot_path, command = sys.argv[1], sys.argv[2], sys.argv[3]
    board = json.load(open(board_path))

    status = board["fields"]["status"]
    agent = board["fields"]["agent"]
    in_progress = status["options"]["In Progress"]
    unassigned = agent["options"]["unassigned"]
    limit = board["wipLimitPerAgent"]

    if in_progress not in command:
        allow()

    match = re.search(r"--id[ =]+([A-Za-z0-9_-]+)", command)
    if not match:
        deny(
            "This call moves a card to In Progress but carries no --id, so the WIP "
            "limit cannot be checked. Use: gh project item-edit --id <item>."
        )
    item_id = match.group(1)

    snapshot = json.load(open(snapshot_path))
    items = snapshot["data"]["node"]["items"]["nodes"]
    by_id = {item["id"]: item for item in items}

    agent_option_ids = set(agent["options"].values())
    tokens = set(command.replace("=", " ").split())
    claimed = next((t for t in tokens if t in agent_option_ids), None)

    if claimed is None and item_id in by_id:
        claimed = selected_option(by_id[item_id], agent["id"])

    if claimed is None or claimed == unassigned:
        deny(
            "A card cannot enter In Progress unassigned. Set the Agent field in the "
            "same call, or claim the card first. The limit is per agent, so an "
            "unassigned card is not countable against anyone."
        )

    agent_names = {v: k for k, v in agent["options"].items()}
    held = [
        item
        for item in items
        if item["id"] != item_id
        and selected_option(item, agent["id"]) == claimed
        and selected_option(item, status["id"]) == in_progress
    ]

    if len(held) < limit:
        allow()

    lines = []
    for item in held:
        content = item.get("content") or {}
        lines.append("  #%s %s" % (content.get("number", "?"), content.get("title", "")))

    deny(
        "WIP limit reached for %s. It already holds %d card(s) In Progress:\n\n%s\n\n"
        "The limit is %d. Move that card to Review or Blocked before claiming another. "
        "The limit exists so work finishes rather than accumulating — see the board README."
        % (agent_names.get(claimed, claimed), len(held), "\n".join(lines), limit)
    )


if __name__ == "__main__":
    main()
