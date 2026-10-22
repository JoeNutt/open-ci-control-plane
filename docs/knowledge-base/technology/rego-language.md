# Rego: the policy language

Reference material for a CI control plane that evaluates declarative policy against the
structure of a pull-request diff. Scope is the **language and policy authoring only** — the
Go embedding API, bundle distribution, and prior art are covered elsewhere.

## How this was verified

Every code example below was executed against a real OPA binary, not transcribed from prose.

```
Version: 1.21.2
Build Commit: 3f7474e6b3c631821d8aac25ca444f7e75a56483
Build Timestamp: 2026-10-12T13:42:18Z
Platform: darwin/arm64
Rego Version: v1
```

Built-in function names and signatures were checked against the authoritative capability
manifest for that release, not against documentation pages:
`https://raw.githubusercontent.com/open-policy-agent/opa/main/capabilities/v1.21.2.json`
(206 built-ins in this release). Where a claim below is **not** verified, it says so
explicitly.

Primary sources:

- Language reference — <https://www.openpolicyagent.org/docs/policy-language>
- Policy reference / built-ins — <https://www.openpolicyagent.org/docs/policy-reference>
- Policy testing — <https://www.openpolicyagent.org/docs/policy-testing>
- Policy performance — <https://www.openpolicyagent.org/docs/policy-performance>
- Style guide — <https://www.openpolicyagent.org/docs/style-guide>
- Errors guide — <https://www.openpolicyagent.org/docs/errors>
- FAQ — <https://www.openpolicyagent.org/docs/faq>
- Keyword reference — <https://www.openpolicyagent.org/docs/policy-reference/keywords/>

---

## 1. The mental model

### Rego is a query language, not a procedural one

Rego descends from Datalog. A Rego module is not a program that runs top to bottom; it is a
set of **rules that define documents**. When you ask OPA for `data.ci.guard.decision`, OPA
searches for variable bindings that satisfy the rules defining that path, and returns the
document those rules generate. Nothing "executes" in source order.

The official framing is worth internalising verbatim:

> When evaluating rule bodies, OPA searches for variable bindings that make all of the
> expressions true. There may be multiple sets of bindings that make the rule body true.

and the two intuitions it gives:

```
rule body  ==  expression-1 AND expression-2 AND ... AND expression-N
rule       ==  rule-name IS value IF body
```

— <https://www.openpolicyagent.org/docs/policy-language#the-basics>

Two consequences follow immediately, and both matter for this project:

**Statement order is semantically irrelevant.** The FAQ states it plainly: "The order in
which statements occur does not matter in Rego. Reorder any two statements and the policy
means exactly the same thing"
(<https://www.openpolicyagent.org/docs/faq#statement-order>). OPA reorders expressions
internally to satisfy safety requirements. Authors who write Rego as if it were a script —
expecting an early check to guard a later one — are writing something that happens to work
rather than something that is correct. (The one deliberate exception is `else`, covered in
§2.)

**A rule body is a conjunction, and a conjunction has no `return`.** There is no early exit
you can author, no mutation, no accumulator variable. If a body needs to compute something
over a collection, it does so with a comprehension or a partial rule, not a loop.

### Undefined is the third truth value, and it is the one that bites

Rego expressions do not merely evaluate true or false. They can be **undefined**, and
undefined propagates.

```rego
package example

v if "hello" == "world"

# also undefined
w if v != true
```

— <https://www.openpolicyagent.org/docs/policy-language#the-basics>

Note what that second rule demonstrates: `w` is not true. `v` is undefined, so a comparison
*against* `v` is undefined too, even though the comparison is `!=`. Undefined is not false;
it is the absence of a value, and almost every operation on an absence is another absence.

When a rule's body is undefined, the rule produces no document. When you query a rule that
produces no document, you get `undefined` — which `opa eval` prints literally and exits zero:

```console
$ opa eval -d conflict.rego -i empty.json 'data.conflict.max_files'
undefined
$ echo $?
0
```

**This is the single most consequential fact for a CI control plane.** An undefined decision
is not a denial and not an error. It is silence. If the host treats silence as "no objection",
then every typo in a field name, every missing webhook attribute, and every unhandled shape
of input becomes an automatic approval.

#### The canonical bug: undefined swallows a `deny`

The style guide names this directly under "Use negation to handle undefined"
(<https://www.openpolicyagent.org/docs/style-guide#use-negation-to-handle-undefined>).
Verified on 1.21.2:

```rego
package gotcha

# BUGGY: if input.reviewers is absent, count() is never reached,
# the body is undefined, and NOTHING is added to the deny set.
deny_buggy contains "no reviewers" if count(input.reviewers) == 0

# CORRECT: push the positive assertion into a helper and negate it.
# The helper fails both when the field is missing and when it is empty.
deny_fixed contains "no reviewers" if not has_reviewers

has_reviewers if count(input.reviewers) > 0
```

Evaluated with `input = {}`:

```json
{
  "deny_buggy": [],
  "deny_fixed": ["no reviewers"]
}
```

The buggy rule produces an empty deny set. A control plane computing
`allow := count(deny) == 0` would wave the PR through. The fix is structural, not a matter of
care: **state the condition you want positively in a helper rule, then negate the helper.**
A negated helper is undefined-safe because `not <undefined>` succeeds.

### How multiple rules with the same name combine

This depends on the rule's *shape*, and conflating the two shapes is a common error.

**Partial rules (sets and objects) union.** The docs call this *incremental definition*:

> A rule may be defined multiple times with the same name. [...] The document produced by
> incrementally defined rules is the union of the documents produced by each individual rule.
> An incrementally defined rule can be intuitively understood as
> `<rule-1> OR <rule-2> OR ... OR <rule-N>`.

— <https://www.openpolicyagent.org/docs/policy-language#incremental-definitions>

This is why `deny contains msg if { ... }` is the workhorse idiom: every author can add a
clause, and the clauses accumulate without coordination.

**Complete rules must agree, or evaluation errors.** A complete rule (one with no key in the
head) can only have one value. If two definitions produce different values for the same
input, OPA raises a runtime error. Verified:

```rego
package conflict

max_files := 10 if input.risk == "high"
max_files := 50 if input.size == "small"
```

```console
$ opa eval -d conflict.rego -i '{"risk":"high","size":"small"}' 'data.conflict.max_files'
1 error occurred: conflict.rego:3: eval_conflict_error: complete rules must not produce multiple outputs
```

Note the stage: this is `eval_conflict_error`, not a compile error. **It is input-dependent.**
The policy compiles, passes `opa check`, may pass its whole test suite, and then fails on the
one PR that happens to satisfy both bodies. Partial objects have the same hazard with a
different message (`eval_conflict_error: object keys must be unique`, verified).

For this project that means: a decision rule whose value varies (a risk score, a routing
target) is a latent production error unless either the bodies are provably disjoint or the
rule is restructured as a partial set that the host reduces.

### Why there are no loops

Because iteration is implicit in the search. A reference containing a variable *is* the
iteration:

```rego
hostnames contains name if {
    name := sites[_].servers[_].hostname
}
```

The `_` is an anonymous variable; OPA finds every binding that satisfies the body, and each
success contributes one element to the set. There is no index to manage and no accumulator.
`some x in xs` is the readable spelling of the same thing, and the style guide prefers it
(<https://www.openpolicyagent.org/docs/style-guide#prefer-some--in-for-iteration>).

### The mistake newcomers make most: variables are existentially quantified

A rule body asks **"does there exist a binding?"**, never "is this true for all?". The docs
devote a whole section to this
(<https://www.openpolicyagent.org/docs/policy-language#universal-quantification-for-all>) and
it is worth reproducing the failure, verified on 1.21.2:

```rego
package g2

files := ["a.go", "infra/iam.tf"]

# WRONG. Reads like "no .tf file was changed". Means "some file is not infra/iam.tf".
no_tf_wrong if {
    some f in files
    f != "infra/iam.tf"
}
```

Result: `no_tf_wrong` is `true` — **even though `infra/iam.tf` is in the list.** It is true
because `"a.go" != "infra/iam.tf"`. The rule can essentially never be false; it is pure noise
that reads like a control.

The three correct forms, all verified:

```rego
# 1. Negate a helper (complement the logic, then complement the check)
no_tf_right if not any_tf
any_tf if {
    some f in files
    f == "infra/iam.tf"
}

# 2. Membership against a set
no_tf_set if not "infra/iam.tf" in {f | some f in files}

# 3. every
no_tf_every if {
    every f in files {
        f != "infra/iam.tf"
    }
}
```

OPA's own linter, Regal, has a dedicated rule for form 1's absence
(<https://www.openpolicyagent.org/projects/regal/rules/bugs/not-equals-in-loop>). Enabling
Regal is the cheapest guardrail available against this class of bug — see §8.

### The second-most-common mistake: `every` over an empty domain is true

```rego
empty_vacuous if {
    every f in [] {
        f == "never"
    }
}
```

Verified: `true`. The docs state it: "If the domain is empty, the overall statement is true"
(<https://www.openpolicyagent.org/docs/policy-language#every-keyword>).

This is correct logic and a trap for diff policy. "Every added route handler must have a
corresponding test" is **vacuously satisfied by a PR that adds no route handlers** — which is
what you want. But "every file in the changeset must be reviewed" is also vacuously satisfied
by an empty or malformed changeset, which is not. Whenever `every` carries a security
decision, assert the domain is non-empty alongside it.

Also note: `every` cannot be negated. The docs: "Negating `every` is forbidden. If you need
to express `not every x in xs { p(x) }` please use `some x in xs; not p(x)` instead."

---

## 2. Core constructs

### The `input` document

`input` is the root of the request-scoped document — for this project, the JSON representation
of the PR diff. It is a plain JSON value; there is no schema enforcement at evaluation time
unless you add one (see §5). References into it are ordinary references, and a reference to a
missing field is **undefined**, not an error:

```rego
targets_main if input.pull_request.base_ref == "main"
```

If `input.pull_request` is absent, this rule is undefined. If the field is misspelled, this
rule is undefined. Both look identical from the outside, which is why §5 and §8 exist.

Alongside `input` sits `data` — the static document tree (external data loaded into OPA, plus
every rule any module defines). A policy reads team ownership tables, path classifications,
and the like from `data`; it reads the PR from `input`.

### Complete rules

A complete rule has a name, no key, and one value. If the value is omitted it defaults to
`true`.

```rego
pi := 3.14159                       # unconditional
allow if input.user.admin           # conditional; value defaults to true
max_files := 500 if input.risk == "low"
```

`allow if { ... }` is shorthand for `allow := true if { ... }` — which the performance docs
confirm, and which matters for early-exit optimisation (§7).

### Partial rules: sets

The `contains` keyword builds a set. This is the shape for accumulating findings.

```rego
package ci.classify

route_handler_paths contains p if {
    some p in input.paths
    glob.match("internal/http/**.go", ["/"], p)
    not endswith(p, "_test.go")
}
```

Each satisfying binding contributes one element. Multiple rules of the same name union.

### Partial rules: objects

An object rule has both a key and a value in the head.

```rego
apps_by_hostname[hostname] := app if {
    some i
    server := sites[_].servers[_]
    hostname := server.hostname
    apps[i].servers[_] == server.name
    app := apps[i].name
}
```

Rule heads may also contain literal references and nested variables, which is a concise way
to build grouped structures:

```rego
package roles

users_by_role[role][id] := user if {
    some user in input.users
    id := user.id
    role := user.role
}

# leaf entries can be partial sets
users_by_country[country] contains user.id if {
    some user in input.users
    country := user.country
}
```

— <https://www.openpolicyagent.org/docs/policy-language#variables-in-rule-head-references>

### Default rules

```rego
default allow := false
```

The default value is used **when all the rules sharing that name are undefined**. The value
must be a scalar, composite, or comprehension containing no variables or references
(comprehensions are permitted because their result is never undefined).

Defaults also work on functions:

```rego
default clamp_positive(_) := 0

clamp_positive(x) := x if x > 0
```

With one sharp caveat the docs flag: a `default` function still fails — does not even return
the default — if an *argument* is undefined, because arguments are evaluated before the call.

**`default` is weaker protection than it looks.** See §8; this is the most important
qualification in this document.

### `some`

`some` declares local variables. Two forms:

```rego
# some ... in: iterate a collection
allow if {
    some role in input.user.roles
    role.id == "admin"
}

# bare some: declare free variables for unification / variable references
tuples contains [i, j] if {
    some i, j
    sites[i].region == "west"
    server := sites[i].servers[j]
    contains(server.name, "db")
}
```

The bare form exists to scope variables locally. Without it, a bare `i` in a body refers to a
*rule* named `i` if one exists in the package — so adding an unrelated rule elsewhere in the
package can silently change another rule's meaning. The docs recommend `some` precisely "to
avoid situations [...] where introduction of a rule inside a package could change behaviour of
other rules". Treat it as mandatory. `opa check --strict` plus Regal will enforce it in
practice.

The three-operand form binds key and value:

```rego
busy contains dir if {
    some dir, n in counts
    n > 1
}
```

### `every`

```rego
every server in site.servers {
    endswith(server.name, "-dev")
}

every i, x in [1, 2, 3] { x - i == 1 }         # array: index, value
every k, v in {"foo": "bar"} { startswith(k, "f") }  # object: key, value
every x in {1, 2, 3} { x != 4 }                 # set
```

"Evaluating `every` does **not** introduce new bindings into the rule evaluation" — variables
bound inside stay inside. Empty domain is true (§1). Cannot be negated.

### Comprehensions

Three forms, all `[head | body]` shaped:

```rego
[ <term> | <body> ]           # array
{ <term> | <body> }           # set
{ <key>: <term> | <body> }    # object
```

```rego
names := [name | sites[i].region == region; name := sites[i].name]
my_set := {e | some e in my_array}
by_dir := {d: members | some p in input.paths; d := dirname(p); members := [...]}
```

Comprehension bodies may close over variables from the enclosing body; OPA reorders the outer
body so those variables are bound first. Object comprehensions cannot produce conflicting
entries, same as object rules.

The style guide prefers a **partial helper rule over a comprehension embedded in a rule body**
when the result is reusable, because a named rule can be queried and debugged on its own
(<https://www.openpolicyagent.org/docs/style-guide#consider-partial-helper-rules-over-comprehensions-in-rule-bodies>).
That advice applies strongly here: `authz_test_paths` as its own rule is testable; the same
comprehension inlined into `decision` is not.

### Built-in functions for string and path matching

All signatures below are from the v1.21.2 capability manifest.

```
glob.match(string, any<null|array[string]>, string) -> boolean
glob.quote_meta(string) -> string
startswith(string, string) -> boolean
endswith(string, string) -> boolean
contains(string, string) -> boolean
indexof(string, string) -> number
split(string, string) -> array[string]
concat(string, any<array[string]|set[string]>) -> string
trim_prefix(string, string) -> string
lower(string) -> string
upper(string) -> string
sprintf(string, array[any]) -> string
strings.count(string, string) -> number
strings.any_prefix_match(any<string|array[string]|set[string]>, any<string|array[string]|set[string]>) -> boolean
strings.any_suffix_match(any<string|array[string]|set[string]>, any<string|array[string]|set[string]>) -> boolean
regex.match(string, string) -> boolean
regex.find_all_string_submatch_n(string, string, number) -> array[array[string]]
regex.globs_match(string, string) -> boolean
```

Collection and set helpers:

```
count(any<string|array[any]|object{any: any}|set[any]>) -> number
sort(any<array[any]|set[any]>) -> array[any]
object.get(object{any: any}, any, any) -> any
object.keys(object{any: any}) -> set[any]
object.subset(any<array|object|set>, any<array|object|set>) -> boolean
object.filter(object{any: any}, any<array|object|set>) -> object{any: any}
object.remove(object{any: any}, any<array|object|set>) -> object{any: any}
intersection(set[set[any]]) -> set[any]     # takes a SET OF SETS
union(set[set[any]]) -> set[any]            # takes a SET OF SETS
walk(any) -> [array[any], any]              # relation
json.match_schema(any<string|object>, any<string|object>) -> [boolean, array[object]]
```

Two notes on these:

- `intersection` and `union` take a **set of sets**, not two sets. For two operands use the
  infix operators instead (below).
- **`set_diff` is marked deprecated** in the v1.21.2 manifest. Use the `-` operator.

Set operations are available as infix operators (verified from the manifest's `infix` field
and by evaluation): `&` is intersection (`and`), `|` is union (`or`), `-` is difference
(`minus`). **These are distinct from the `and`/`or` *keywords* described below** — a genuine
naming collision in Rego worth knowing about.

### String interpolation

Added in **OPA 1.12.0**
(<https://github.com/open-policy-agent/opa/blob/main/CHANGELOG.md> § 1.12.0). A `$`-prefixed
string takes `{...}` template expressions:

```rego
violations contains $"auth middleware modified: {path}" if {
    some path in classify.auth_middleware_paths
}
```

The behavioural difference from `sprintf` is the point, and it favours interpolation for this
project: **an undefined template expression emits the literal string `<undefined>` rather than
making the whole rule undefined.** `sprintf` with an undefined argument produces no result at
all — meaning a missing field silently deletes a finding from your deny set. The docs
therefore call interpolation "the recommended alternative"
(<https://www.openpolicyagent.org/docs/policy-language#undefined-values>).

Escape a literal brace with `\{`.

### `with` — the mocking and override keyword

```
<expr> with <target-1> as <value-1> [with <target-2> as <value-2> [...]]
```

Targets must be references into `input`, into `data`, or to functions (built-in or
user-defined). `with` modifies **only the attached expression**.

```rego
result := true if {
    allow with input as {"user": "charlie", "method": "GET"}
          with data.roles as {"dev": ["charlie"]}
          with time.weekday as "Sunday"
}
```

For function targets, the replacement may be a value, another function, another built-in, or a
rule. Arity must match. "Replacement functions can call the function they're replacing
**without causing recursion**." Replacing `internal.*`, `rego.metadata.*`, `eq`, or relations
like `walk` is not allowed.

`with` is how policy gets tested (§6). It is also the mechanism that makes `http.send` and
`time.now_ns` testable — relevant if policy ever reaches outside, though for a sub-millisecond
budget it should not.

### `else`

```rego
authorize := "allow" if {
    input.user == "superuser"
} else := "deny" if {
    input.path[0] == "admin"
    input.source_network == "external"
}
```

`else` is the one place Rego has ordered evaluation: "Rules grouped together with the `else`
keyword are evaluated until a match is found. Once a match is found, rule evaluation does not
proceed to rules further in the chain." The docs recommend using it "sparingly to avoid
tightly coupled rules". For an end-user-authored policy set, `else` chains are a readability
and composition liability — prefer disjoint bodies or a partial set the host reduces.

### Equality operators — which to use

| Operator | Name | Use for |
|---|---|---|
| `:=` | assignment | declaring a new local; errors if the name is already bound |
| `==` | comparison | checking two values are the same; both sides must be safe |
| `=` | unification | solving for variables on either side; destructuring against a pattern |

The style guide says: "Don't use unification operator for assignment or comparison"
(<https://www.openpolicyagent.org/docs/style-guide#dont-use-unification-operator-for-assignment-or-comparison>).

But there is a real case where `=` is **required**, which I hit while verifying §5 — you
cannot `:=` into a pattern containing literals:

```rego
# rego_compile_error: cannot assign to boolean
valid if { [true, _] := json.match_schema(input, schema) }

# correct
valid if { [true, _] = json.match_schema(input, schema) }
```

### Language features newer than you may expect

These are version-gated and easy to get wrong. Verified against 1.21.2 by evaluation.

| Feature | Import required | Available since |
|---|---|---|
| `if`, `contains`, `in`, `every` | none (standard since v1.0) | OPA 1.0.0 |
| String interpolation `$"..."` | none | OPA 1.12.0 |
| Improved `not` semantics, `not { ... }` bodies | `import future.keywords.not` | OPA 1.17.0 |
| `and` / `or` keywords | `import future.keywords.and` / `.or` | OPA 1.20.0 |
| `import rego.v2` (enables all three at once) | — | **Unreleased as of 1.21.2** |

I verified the last row directly, because it is a trap: the current published documentation
for the `not` keyword and the `import` keyword both reference `rego.v2` as though it were
available, and the OPA CHANGELOG places it under **Unreleased**. On 1.21.2 it is a parse
error:

```console
$ opa check v2.rego
1 error occurred during loading: v2.rego:2: rego_parse_error: invalid import `rego.v2`, must be `rego.v1`
```

Until a release ships it, use the three `future.keywords.*` imports individually.

#### `future.keywords.not` deserves its own note — it changes fail-open to fail-closed

This is the most security-relevant language change in recent OPA, and it directly concerns
this project. Without the import, OPA expands a negated composite expression into
sub-expressions evaluated *before* the `not`; if any sub-expression is undefined, **the whole
rule fails rather than the `not` succeeding**. With the import, the expansion is wrapped in an
implicit body, so an undefined sub-expression makes the body unsatisfiable and the `not`
succeeds.

I reproduced the documented example to measure the actual difference:

```rego
package legacy

groups := {"admin": ["alice"], "user": ["bob"]}
lookup(user) := group if { some group, members in groups; user in members }
admin(group) if group in ["admin", "sudo"]

restricted if not admin(lookup(input.user))
```

| `input.user` | legacy `restricted` | with `future.keywords.not` |
|---|---|---|
| `alice` (admin) | undefined | undefined |
| `bob` (user) | `true` | `true` |
| `cesar` (**unknown**) | **undefined** | **`true`** |

An unknown user — the case a security control exists for — is *not restricted* under legacy
semantics and *is restricted* with the import. The docs' recommendation is unambiguous:
"Users are recommended to use the `future.keywords.not` [...] import whenever using the `not`
keyword." For a policy set that must fail closed, treat the import as compulsory and lint for
its presence.

The import also unlocks explicit negation bodies, which are genuinely useful for diff policy:

```rego
import future.keywords.not

deny contains $"server {server.name} is misconfigured" if {
    some server in input.servers
    not {
        # if ANY of these fails, the not succeeds
        listener := server.listener
        listener.port == 443
        listener.protocol == "tcp"
    }
}
```

Variables declared inside the body are scoped to it. Assignment is allowed only inside an
explicit body — `not listener := server.listener` is a compile error.

#### `and` / `or` keywords

```rego
import future.keywords.and
import future.keywords.or

allow if {
    input.method == "GET"
    input.user.admin or input.user.owner and input.resource.public
}
```

`and` binds tighter than `or`; `not` binds tighter than both. An `and`/`or` expression
succeeds or fails and **never produces a value** — you cannot assign it, pass it to a
function, or use it as a comprehension head. Operands can read enclosing variables but cannot
create new ones for the rest of the rule; brace an operand to give it its own scope.
Evaluation is short-circuiting, and when both sides of an `or` succeed you still get a single
result.

The CHANGELOG is explicit that `or` does not replace incremental rules: "`or` is not a
replacement for expressing disjunction with incremental rules, which remains the idiom when
the disjunction needs to contribute a value."

---

## 3. Idioms for diff-shaped input

This section uses one concrete input shape throughout. It is the shape §5 argues for.

```json
{
  "schema_version": 1,
  "pull_request": {
    "number": 412,
    "base_ref": "main",
    "author": {"login": "jo", "association": "MEMBER"},
    "labels": ["backend"]
  },
  "paths": [
    "internal/http/routes.go",
    "internal/auth/middleware.go",
    "infra/iam.tf"
  ],
  "files": [
    {"path": "internal/http/routes.go", "status": "modified",
     "additions": 31, "deletions": 2,
     "added_lines": ["\tmux.Handle(\"POST /v1/payouts\", h.createPayout)"]},
    {"path": "internal/auth/middleware.go", "status": "modified",
     "additions": 4, "deletions": 1,
     "added_lines": ["\tif tok == \"\" { return nil }"]},
    {"path": "infra/iam.tf", "status": "added",
     "additions": 12, "deletions": 0,
     "added_lines": ["  actions = [\"s3:*\"]"]}
  ]
}
```

### Glob matching over paths: the delimiter is the whole game

`glob.match(pattern, delimiters, string)`. The second argument is an **array of
single-character delimiters**, or `null` for none, or `[]` to default to `"."`.

For file paths you almost always want `["/"]`. Verified behaviour on 1.21.2:

| Call | Result |
|---|---|
| `glob.match("*.go", ["/"], "cmd/server/main.go")` | `false` — `*` does not cross `/` |
| `glob.match("**.go", ["/"], "cmd/server/main.go")` | `true` |
| `glob.match("internal/**", ["/"], "internal/auth/middleware.go")` | `true` |
| `glob.match("infra/*.tf", ["/"], "infra/iam.tf")` | `true` |
| `glob.match("infra/**.{tf,tfvars}", ["/"], "infra/mod/iam.tf")` | `true` |
| `glob.match("*_test.go", null, "internal/http/routes_test.go")` | `true` — no delimiter, `*` crosses `/` |

Documented pitfalls, from
<https://www.openpolicyagent.org/docs/policy-reference/builtins/glob>: delimiters must be
**single characters** — a multi-character or empty delimiter returns **undefined** (or an
error under `--strict-builtin-errors`). An empty array `[]` defaults to `"."`, which for file
paths is almost certainly a bug waiting to happen: `glob.match("*.go", [], p)` splits on dots,
not slashes.

#### A real trap: `**/` does not match the top level

Verified:

```rego
glob.match("**/*_test.go", ["/"], "main_test.go")      # false
glob.match("**/*_test.go", ["/"], "a/b/main_test.go")  # true
glob.match("{**/,}*_test.go", ["/"], "main_test.go")   # true
```

A policy requiring "an authorization test was touched" written as `**/*_test.go` **will not
see a test file at the repository root** and will block a PR that did the right thing. Prefer
`endswith(p, "_test.go")` when you only care about the suffix, and reserve globs for cases
where the directory genuinely matters. Where you need both, `{**/,}` is the documented
alternation that covers zero-or-more leading segments.

Also note: the indexer does not index `**` patterns at all (§7), so super-globs cost you both
correctness clarity and speed.

### Classify once, decide separately

The structure that works is a classification layer over the raw diff and a decision layer over
the classifications. Each classification is an independently testable named rule.

```rego
package ci.classify

paths := {p | some p in input.paths}

route_handler_paths contains p if {
    some p in paths
    glob.match("internal/http/**.go", ["/"], p)
    not endswith(p, "_test.go")
}

authz_test_paths contains p if {
    some p in paths
    endswith(p, "_test.go")
    glob.match("internal/{auth,authz}/**", ["/"], p)
}

auth_middleware_paths contains p if {
    some p in paths
    glob.match("internal/auth/**.go", ["/"], p)
    not endswith(p, "_test.go")
}

ci_config_paths contains p if {
    some p in paths
    glob.match(".github/workflows/*", ["/"], p)
}

iac_paths contains p if {
    some p in paths
    strings.any_suffix_match(p, [".tf", ".tfvars"])
}
```

Note `paths` is converted to a **set** on the first line. The style guide's "Prefer sets over
arrays (where applicable)" applies: set lookups are O(1), array lookups O(n)
(<https://www.openpolicyagent.org/docs/style-guide#prefer-sets-over-arrays-where-applicable>).
With a few hundred changed files, this is the difference measured in §7.

### Checking for the *absence* of a file in the changeset

This is the project's key pattern and Rego's classic gotcha. The wrong form is the
existentially-quantified `!=` from §1. There are three correct forms; all verified.

```rego
# 1. count a comprehension / partial rule
no_authz_test if count(classify.authz_test_paths) == 0

# 2. not + membership (when you have a specific path in mind)
no_changelog if not "CHANGELOG.md" in classify.paths

# 3. not + helper rule (the general case; undefined-safe)
no_authz_test_2 if not any_authz_test
any_authz_test if {
    some p in classify.paths
    endswith(p, "_test.go")
    glob.match("internal/{auth,authz}/**", ["/"], p)
}
```

Form 3 is the one to teach end users, because it is the only one that stays correct when the
input field itself is missing — `not any_authz_test` succeeds whether `input.paths` is empty
*or* absent. Forms 1 and 2 are fine here only because `classify.paths` is a comprehension
(never undefined); written directly against `input.paths` they would inherit the undefined
problem from §1.

### Set operations across file lists

Verified:

```rego
package setops

changed := {p | some p in input.paths}

touched_both := route_handlers & authz_tests          # intersection
all_interesting := route_handlers | authz_tests       # union
handlers_without_tests := route_handlers - authz_tests # difference

# "is the changeset confined to an allowed set?"  object.subset(super, sub)
allowed := {"docs/README.md", "CHANGELOG.md"}
docs_only if object.subset(allowed, changed)
docs_only_via_diff if count(changed - allowed) == 0
```

`object.subset(a, b)` is true when `a` is a superset of `b` — argument order verified by
evaluation (`object.subset(allowed, changed)` is `true` when `allowed ⊇ changed`). The
`count(changed - allowed) == 0` form reads more obviously to most authors and is worth
preferring for that reason alone.

### The four example policies

#### "Adds a route handler but touches no authorization test → block"

```rego
violations contains $"route handler added in {path} with no change to an authorization test" if {
    some path in classify.route_handler_paths
    count(classify.authz_test_paths) == 0
}
```

#### "Modifies auth middleware at all → require human sign-off"

```rego
violations contains $"auth middleware modified: {path}" if {
    some path in classify.auth_middleware_paths
}
```

#### "Changes CI config → route to the platform team"

Routing is not a denial, so it belongs in a separate document the host reads independently:

```rego
package ci.routing

reviewers contains "platform" if count(data.ci.classify.ci_config_paths) > 0
reviewers contains "security" if count(data.ci.classify.auth_middleware_paths) > 0
reviewers contains team if {
    some path in data.ci.classify.paths
    some pattern, team in data.ownership.by_glob
    glob.match(pattern, ["/"], path)
}
```

The last clause joins the diff against an ownership table in `data`. Note that this is a
cross-product over (paths × patterns) — bounded by the size of the ownership table, which is
the right trade, but see §7 before letting that table grow.

#### "Adds an IaC resource with a wildcard IAM action → block"

This needs line content, not just paths. Verified against the input above:

```rego
iam_wildcards contains {"path": f.path, "line": line} if {
    some f in input.files
    endswith(f.path, ".tf")
    some line in f.added_lines
    regex.match(`"[a-z0-9]+:\*"`, line)
}
```

Output:

```json
[{"line": "  actions = [\"s3:*\"]", "path": "infra/iam.tf"}]
```

Two cautions. First, use **raw strings** (backticks) for regex patterns — the style guide says
so (<https://www.openpolicyagent.org/docs/style-guide#use-raw-strings-for-regex-patterns>) and
without them every backslash needs doubling. Second, this is grepping a diff, not parsing HCL.
It will miss a wildcard spread across lines and will fire on a comment. That is a limitation
of the *input design*, not of Rego: if wildcard IAM detection matters, the control plane should
parse the Terraform and hand policy a structured `resources` array. §5 returns to this.

---

## 4. Designing the `input` document

This is the project's most consequential early decision, and it is consequential for two
independent reasons: it is a **published contract** you cannot change without breaking every
team's policy, and it **determines the performance envelope** more than any amount of Rego
tuning will.

### It is a contract, so version it

Put `schema_version` in the document from day one, and make the host refuse to evaluate
against a policy bundle that expects a different major version. Renaming `paths` to
`changed_paths` after three teams have written policy is not a refactor; it is a migration.

The corollary is that **everything in the document is permanent**. Do not expose fields
"because they were in the webhook". A GitHub webhook payload is enormous, deeply nested,
inconsistently populated, and shaped by GitHub's needs rather than policy's. Passing it
through verbatim publishes GitHub's API as your policy contract and guarantees you inherit
every one of its breaking changes.

### Flat and pre-computed beats nested and raw — and the gap is measurable

Rego is a poor place to reshape data. It has no mutation, so every regrouping is a
comprehension, and comprehensions over a few hundred files are where the latency budget goes.

I measured the same question — "which directories have more than 60 changed files?" — answered
three ways over a 500-file diff. `opa bench`, OPA 1.21.2, Darwin arm64:

| Implementation | ns/op | B/op |
|---|---:|---:|
| Group-by in Rego, comprehension-indexed | 2,727,709 (2.73 ms) | 1,583,088 |
| Group-by in Rego, accidental cross-product | 345,823,444 (**346 ms**) | 69,369,565 |
| Host pre-computes `input.by_dir`; policy reads it | 12,241 (**12 µs**) | 6,196 |

The pre-computed field is **~220× faster** than the well-written Rego version and ~28,000×
faster than the naive one. A simple count over a pre-computed list (`count(input.kinds.test)`)
measured 6,442 ns. An indexed `allow`-shaped decision measured 6,933 ns.

Those numbers set the design rule plainly: **anything the host can compute in Go for free,
the host should compute.** Go iterates a 500-element slice in microseconds; Rego's search
machinery does not.

### What a good input document looks like

```json
{
  "schema_version": 1,
  "pull_request": {
    "number": 412,
    "base_ref": "main",
    "draft": false,
    "author": {"login": "jo", "association": "MEMBER"},
    "labels": ["backend"]
  },
  "paths": ["internal/http/routes.go", "internal/auth/middleware.go", "infra/iam.tf"],
  "files": [
    {"path": "internal/http/routes.go", "status": "modified",
     "additions": 31, "deletions": 2, "added_lines": ["..."]}
  ],
  "totals": {"files": 3, "additions": 47, "deletions": 3},
  "kinds": {
    "test": [],
    "iac": ["infra/iam.tf"],
    "ci_config": [],
    "generated": []
  },
  "by_dir": {
    "internal/http": ["internal/http/routes.go"],
    "internal/auth": ["internal/auth/middleware.go"],
    "infra": ["infra/iam.tf"]
  }
}
```

The design principles behind it:

1. **A flat `paths` array alongside the detailed `files` array.** Most policies only ask "was
   anything matching X touched?". Making them walk `files[_].path` to answer that is a tax on
   every author and every evaluation. The duplication is deliberate and cheap.
2. **Pre-computed classification buckets (`kinds`).** The host knows which paths are tests,
   IaC, CI config, or generated — it has the repo conventions. Computing those once in Go and
   publishing them means authors write `count(input.kinds.test) == 0` instead of each
   reinventing a glob, each slightly differently, each slightly wrong (recall `**/` from §3).
   This is the field that most reduces broken policy.
3. **Pre-computed aggregates (`totals`).** `count()` over a list is cheap, but `totals` lets
   simple size policies be indexable scalar comparisons rather than function calls (§7).
4. **Pre-computed groupings (`by_dir`) only where a policy actually needs them.** This is the
   220× field. Add groupings on demand, not speculatively — each one is contract surface
   forever.
5. **Pre-parsed structure where grepping would be wrong.** If "wildcard IAM action" is a real
   control, publish `input.iac_resources` with parsed actions rather than letting every team
   write a regex over `added_lines`. Policy should match on facts, not re-derive them.
6. **No nulls, no optional-by-omission fields on anything load-bearing.** An absent field is
   undefined, and undefined fails open (§1). Emit `[]`, `0`, `false` — a present empty value is
   something policy can reason about; an absent one is a silent hole. This single rule
   eliminates most of the §1 bug class before authors can write it.
7. **Stable ordering.** If `paths` is sorted, policies that compare or hash it are
   reproducible, and tests are stable.

### What a bad input document looks like

- The raw GitHub webhook payload, passed through.
- A single deeply-nested tree (`input.pr.diff.files[].meta.classification.kind`) — every
  reference is a longer chain, every chain has more places to be undefined, and `opa check
  --schema` is the only thing standing between an author and a silent typo.
- Fields whose presence varies by event type, so policy must defensively test for existence
  before every use.
- Diff *text* as the only representation of content, forcing regex where structure was
  available.
- No version field.

### Publish a JSON Schema with the document — it is the best guardrail available

This is the highest-value finding in this document for authoring ergonomics. OPA can
type-check policy against a JSON Schema for `input` **at compile time**, and it catches the
exact failure that otherwise fails open.

```json
// schemas/pr_diff.json
{
  "type": "object",
  "properties": {
    "schema_version": {"type": "integer"},
    "paths": {"type": "array", "items": {"type": "string"}},
    "pull_request": {
      "type": "object",
      "properties": {"base_ref": {"type": "string"}}
    }
  }
}
```

```rego
# METADATA
# scope: package
# schemas:
#   - input: schema.pr_diff
package typo2

targets_main if input.pull_request.base_branch == "main"   # typo: base_ref
```

Verified on 1.21.2:

```console
$ opa check --schema schemas/ p.rego
1 error occurred: p.rego:7: rego_type_error: undefined ref: input.pull_request.base_branch
	input.pull_request.base_branch
	                   ^
	                   have: "base_branch"
	                   want (one of): ["base_ref"]

$ opa check p.rego      # without the schema
$ echo $?
0
```

Without the schema the typo compiles clean and evaluates to undefined forever. With it you get
the error *and a suggestion*. `opa test --schema schemas/` enforces the same thing (verified),
so it lands in CI for free. A single-file schema can be passed as `--schema schemas/input.json`
and applies to `input` automatically; a directory plus `METADATA` annotations lets different
packages declare different schemas.

**Know the limits.** The docs list features the schema type-checker does not support
(<https://www.openpolicyagent.org/docs/policy-language#limitations>): `additionalProperties`,
`patternProperties`, `additionalItems`, `contains`, `oneOf`, `not`, `enum`, and
`if`/`then`/`else`. It is a *reference* checker, not a validator — it tells you a field exists
and roughly what type it is. Also note that when a schema is fully specified, OPA treats it as
closed "even if `additionalProperties` is set to `true`", which is what makes typo detection
work but means an intentionally open-ended field must be declared as a bare `object` to avoid
false errors.

For actual runtime validation of the input shape, `json.match_schema` is a full JSON Schema
validator and is cheap — 30,806 ns/op measured on the schema above. It supports `const` and
`required`, which the compile-time checker ignores:

```rego
package contract

valid if {
    [true, _] = json.match_schema(input, schema)
}

errors := es if {
    [false, es] = json.match_schema(input, schema)
}
```

Verified output on malformed input:

```json
{"errors": [
  {"desc": "Invalid type. Expected: array, given: string",
   "error": "paths: Invalid type. Expected: array, given: string",
   "field": "paths", "type": "invalid_type"},
  {"desc": "schema_version does not match: 1",
   "error": "schema_version: schema_version does not match: 1",
   "field": "schema_version", "type": "const"}
]}
```

That said, validating input inside Rego duplicates a check the host should already have done
at the boundary. Prefer validating in Go when constructing the document, and use `--schema`
for the authoring-time guardrail. `json.match_schema` is the fallback for when the policy
bundle must defend itself against an older host.

---

## 5. Policy testing

### Mechanics

Tests are ordinary Rego rules whose names begin with `test_`. Convention — not a requirement —
is to put them in a package suffixed `_test`.

```rego
package ci.guard_test

import data.ci.guard

pr(files) := {"paths": [f | some f in files], "files": []}

test_auth_middleware_change_blocks if {
    d := guard.decision with input as pr(["internal/auth/middleware.go"])
    d.effect == "block"
    "auth middleware modified: internal/auth/middleware.go" in d.reasons
}

test_docs_only_change_allows if {
    d := guard.decision with input as pr(["docs/README.md"])
    d.effect == "allow"
    d.reasons == []
}
```

```console
$ opa test v/ -v
data.ci.guard_test.test_auth_middleware_change_blocks: PASS (452.167µs)
...
PASS: 6/6
```

Result semantics, from
<https://www.openpolicyagent.org/docs/policy-testing#test-results>:

| Outcome | Meaning |
|---|---|
| `PASS` | rule evaluated to `true` |
| `FAIL` | rule was **undefined** or produced a non-`true` value |
| `ERROR` | rule hit a runtime error (e.g. divide by zero) |
| `SKIPPED` | rule name is prefixed `todo_` instead of `test_` |

Note that `FAIL` conflates "returned false" with "was undefined". That is the same conflation
§1 warns about, now in your test output — which is exactly why `--var-values` exists.

### Mocking with `with`

`with` is the whole mocking story, and it is unusually capable: it replaces `input` (or any
sub-path of it), `data` (base or virtual documents), and functions.

```rego
# whole input
test_x if { guard.decision with input as {...} }

# a sub-path of input, leaving the rest intact
test_y if { authz.allow with input.headers["x-token"] as "my-jwt" }

# base data
test_z if { authz.allow with data.roles as {"admin": ["alice"]} }

# a virtual document (another rule!)
test_replace_rule if { authz.allow1 with authz.allow2 as true }

# a built-in, replaced by a user-defined function
mock_decode_verify("my-jwt", _) := [true, {}, {}]
mock_decode_verify(x, _)        := [false, {}, {}] if x != "my-jwt"

test_allow if {
    authz.allow with io.jwt.decode_verify as mock_decode_verify
}

# a built-in, replaced by a bare value (returns it regardless of arguments)
test_allow_value if { authz.allow with io.jwt.decode_verify as [true, {}, {}] }
```

Constraints (<https://www.openpolicyagent.org/docs/policy-testing#data-and-function-mocking>):
`internal.*`, `rego.metadata.*`, `eq` and relations (`walk`) cannot be replaced; arity must
match; a replacement may call the function it replaces without recursing.

The ability to mock a *rule* (`with authz.allow2 as true`) is what makes the
classify/decide split from §3 testable in isolation: you can test `decision` against synthetic
classifications without constructing a diff that produces them.

### Parameterized tests

A test rule can carry a table. Test-case names go in the head reference as variables:

```rego
package ci.guard_test

test_cases[note] if {
    some note, tc in {
        "ci config": {"paths": [".github/workflows/ci.yaml"], "exp": "allow"},
        "iam":       {"paths": ["infra/iam.tf"],              "exp": "allow"},
    }
    d := guard.decision with input as pr(tc.paths)
    d.effect == tc.exp
}
```

Each case reports individually:

```console
data.example_test.test_concat: FAIL (263.375µs)
  empty + empty: PASS
  empty + filled: PASS
  filled + filled: FAIL
```

Cases can be nested (`test_sign_token[note][alg]`) and can be loaded from JSON or YAML data
files rather than inlined, which is the right shape for a growing table of diff fixtures:

```rego
test_concat[note] if {
    some note, tc in data.test_cases
    ...
}
```

```console
$ opa test file_example_test.rego file_example_test.yaml
```

### `--var-values`: read this flag's output instead of guessing

When a test fails, `--var-values` prints the failing expression with every variable's value
annotated beneath it. Verified:

```console
$ opa test vv_test.rego --var-values
data.vv_test.test_bad: FAIL (216.458µs)

  vv_test.rego:3:
                                       f([n]) == 2
                                       |  |
                                       |  1
                                       1
```

This is the fastest way to resolve the FAIL-means-undefined-or-false ambiguity, and it should
be the default flag in the project's test task.

### `print()` for development

Verified working in both `opa eval` and `opa test -v`, and verified that **output goes to
stderr**, so it never pollutes a decision on stdout:

```rego
d contains p if {
    some p in input.paths
    print("considering:", p, "is_test:", endswith(p, "_test.go"))
    endswith(p, ".go")
}
```

```console
$ opa test p.rego p_test.rego -v
data.dbg_test.test_d: PASS (2.242292ms)

  considering: a.go is_test: false
  considering: b_test.go is_test: true
  considering: c.md is_test: false
```

`print` is in the capability manifest as `print(, any...) -> void`. (In an embedded Go host it
requires a print hook to be wired up — that is an embedding concern, out of scope here, but
worth knowing before telling authors `print` works everywhere.)

### Coverage

```console
$ opa test v/ --coverage --format=json
```

Verified output shape, trimmed:

```json
{"files": {
  "v/classify.rego":    {"covered_lines": 12, "coverage": 100},
  "v/policy.rego":      {"not_covered": [{"start": {"row": 18}, "end": {"row": 18}}],
                         "coverage": 92.3}
}, "coverage": 65.8}
```

The semantics matter more than the number
(<https://www.openpolicyagent.org/docs/policy-testing#coverage>):

> - If the line refers to the head of a rule, the body of the rule was never true.
> - If the line refers to an expression in a rule, the expression was never evaluated.

So an uncovered `violations contains ...` head means **no test ever made that violation
fire**. For a policy whose entire job is to produce violations, uncovered rule heads are the
coverage signal that actually matters — far more than a percentage. In my verification run,
`policy.rego:18` showed uncovered until I added the test that actually triggers the
route-handler block; the number went to 100 and the gap was real.

Two refinements the docs flag. First, coverage can be distorted by evaluation-time
optimisation: "an evaluation-time optimization determined some path unnecessary for
evaluation". Rule indexing and early exit (§7) can both skip a path that *was* logically
reached. Second, OPA now annotates why:

> - `index_excluded`: the rule indexer excluded the path without attempting it.
> - `early_exit`: an early-exit optimization skipped the path without attempting it.

controlled by `--coverage-runs` (available on both `opa test` and `opa eval`; both kinds run
by default; an empty list disables the extra passes).

`--threshold N` exits non-zero below N% coverage — verified `exit=0` at `--threshold 100` on a
fully covered suite.

### `opa test` flags, verified from source

Read from `cmd/test.go` at the v1.21.2 tree, so these are exact:

| Flag | Short | Effect |
|---|---|---|
| `--verbose` | `-v` | verbose reporting |
| `--coverage` | `-c` | report coverage (overrides debug tracing) |
| `--coverage-runs` | | supplementary coverage passes (`index_excluded`, `early_exit`); requires `--coverage`; empty list disables |
| `--threshold` | | fail below this coverage % |
| `--run` | `-r` | run only tests matching an RE2 regex |
| `--fail-on-empty` | | fail if no test ran |
| `--var-values` | | annotate failures with variable values |
| `--bench` | | benchmark the unit tests |
| `--parallel` | `-p` | parallel tests (default = NumCPU; benchmarks always sequential) |
| `--timeout` | | test timeout (default 5s; 30s when benchmarking) |
| `--exit-zero-on-skipped` | `-z` | skipped tests exit 0 |
| `--watch` | `-w` | re-run on file change |
| `--format` | | `pretty` (default) or `json` |
| `--schema` | | type-check against JSON Schema (see §4) |
| `--explain` | | explain mode; implies `--verbose` |

`--fail-on-empty` verified: `exit=1` when `-r` matches nothing, `exit=0` without the flag.
Put it in CI — a typo'd `--run` pattern otherwise reports success having tested nothing.

### What a well-tested policy suite looks like here

Given the project tests first as a matter of discipline, the shape that pays off:

1. **One test per classification rule**, driven by a path list, including the boundary cases
   globs get wrong: repository-root files, nested files, files whose names merely contain the
   pattern (`routes_test.go.bak`), and paths with unusual characters.
2. **A denial test for every control.** A test proving auth-middleware changes block proves
   nothing about whether anything *else* blocks. Every `violations` clause needs a test that
   fires it and a test that confirms it stays quiet on an unrelated diff.
3. **A fail-closed test for every malformed input**: `{}`, `{"paths": []}`,
   `{"paths": "not-an-array"}`, missing `schema_version`. These are the tests that catch the
   §1 bug class, and they are the tests teams will not write unless the project provides them
   as a template. My own example policy failed `test_empty_input_fails_closed` on the first
   run — see §8 for what was wrong and why `default` did not save it.
4. **A table-driven suite over committed diff fixtures**, using parameterized tests with YAML
   data, so adding a regression case is adding a few lines of YAML rather than a new rule.
5. **Coverage gated on uncovered rule heads**, not on a percentage.
6. **`opa check --strict --schema schemas/` and `opa fmt --fail` in CI** alongside the tests.

---

## 6. Performance characteristics of the language

The budget here is well under 500 ms, ideally sub-millisecond. That is comfortably achievable
— but only for policy written in the shapes OPA optimises, and the penalty for the wrong shape
is three orders of magnitude.

All measurements: `opa bench`, OPA 1.21.2, Darwin arm64, 500-path synthetic diff. These are
relative indicators on one machine, not portable absolutes.

### Measured baseline

| Operation | ns/op |
|---|---:|
| Indexed `allow`-shaped decision (3 rules, scalar comparisons) | 6,933 |
| `count()` over a pre-computed 500-element list | 6,442 |
| Lookup in a pre-computed object (`input.by_dir`) | 12,241 |
| Linear scan of 500 paths with `endswith` | 141,172 |
| Set-membership absence check over 500 paths | 1,242,149 |
| Group-by over 500 paths (comprehension-indexed) | 2,727,709 |
| Nested scan: for each path, scan all paths | 221,094,150 |
| Cross-product pairs over 500 paths | 345,823,444 |

Read the gap between the last two rows and the rest. A single accidentally quadratic rule
turns a 7 µs decision into a 346 ms one, allocating 69 MB per evaluation. That busts the
budget on one PR — and a policy author will not notice while testing against a three-file
fixture.

### The linear fragment

OPA documents a fragment of Rego "engineered to evaluate in near constant time", where
"adding more rules to the policy will not significantly increase the evaluation time"
(<https://www.openpolicyagent.org/docs/policy-performance#high-performance-policy-decisions>).
The property is that each local variable can take only one value, so evaluation is a walk over
the rule bodies rather than a search.

```rego
allow if {
    some user
    input.method == "GET"
    input.path = ["accounts", user]
    input.user == user
}
```

### Rule indexing: which statement shapes are cheap

Within the linear fragment, OPA indexes rules so that most are never evaluated at all. This is
documented in unusual detail at
<https://www.openpolicyagent.org/docs/policy-performance#use-indexed-statements>. The parts
relevant to diff policy:

**Equality.** One side a non-nested variable-free reference, the other a variable, scalar, or
array.

| Expression | Indexed |
|---|---|
| `input.x == "foo"` | yes |
| `input.x.y == "bar"` | yes |
| `input.x == ["foo", i]` | yes |
| `input.x[i] == "foo"` | yes (ground prefix `input.x`) |
| `input.x[i].y == "foo"` | **no** — variable is not the last element |
| `input.x[i] == ["foo"]` | **no** — non-scalar value on a reference containing a variable |

**Prefix and suffix.** `startswith`, `endswith`, `strings.any_prefix_match`,
`strings.any_suffix_match` are indexed when the search operand is a variable-free
non-nested reference and every base string is known at compile time.

| Expression | Indexed |
|---|---|
| `startswith(input.path, "/api")` | yes |
| `strings.any_suffix_match(input.path, [".go", ".rego"])` | yes — each base is an alternative |
| `startswith(input.path, input.prefix)` | **no** — base unknown until evaluation |
| `startswith(input.path[i], "/api")` | **no** — search contains a variable |
| `x := startswith(input.path, "/api"); x == true` | **no** — result is captured |

The docs note prefixes are held in a radix trie, so "a rule set with ten prefixes and one with
ten thousand cost the same to look up". This is excellent news for path policy: a large
allowlist of path prefixes is free to look up.

**Glob.** Indexed only when the pattern is recognised — plain `*` forming a complete
delimiter-separated segment. Not indexed for `**`, for character classes, when the match
operand contains a variable, or when the delimiter is `null`.

| Expression | Indexed |
|---|---|
| `glob.match("foo:*:bar", [":"], input.x)` | yes |
| `glob.match("foo:**:bar", [":"], input.x)` | **no** — contains `**` |
| `glob.match("a*", [":"], input.x)` | **no** — `*` embedded in a segment |
| `glob.match("foo:*:bar", null, input.x)` | **no** — null delimiter |

Note the tension with §3: the `**` patterns most natural for file paths are exactly the ones
that do not index. For hot-path decisions prefer `startswith` / `endswith` /
`strings.any_suffix_match`, which index, and reserve globs for the classification layer where
you are iterating anyway.

**Membership.** `value in collection` indexes when the value is a scalar and the collection is
a variable-free reference or a literal set/array/object. But the docs warn: "A lookup for
`"admin" in input.roles` walks `input.roles`, so its cost grows with the collection. For
large collections, prefer an object and `input.roles["admin"]`."

**Bare references into base data.** `allow if data.groups.admins.members[input.subject]` is
indexed as a single hash lookup against the collection, read at lookup time (so it sees
current data and honours `with`). Only objects index this way — not sets (JSON has none) and
not arrays.

**Negation is never indexed.** `not (input.x == 1 and input.y == 2)` — no.

**References rooted at a local resolve.** `x := input; x.foo == "a"` indexes exactly like
`input.foo == "a"`, and chains resolve too. But `x := some_rule; x.foo == "a"` does not — the
indexer only looks at base documents — and neither does a function argument (`f(x) if x.foo ==
"a"`). That last one is a real constraint on the style guide's advice to prefer function
arguments over `input`: it is better design and it forfeits indexing.

### Early exit

> 1. A set of complete document rules that only have one, ground value.
> 2. A set of function rules that only have one, ground value.

qualify for early exit, and "iterations inside that rule will be **cancelled** as soon as one
binding matches the rule body"
(<https://www.openpolicyagent.org/docs/policy-performance#early-exit-in-rule-evaluation>).

Eligible: `p if {...}`, `q := 123 if {...}`, `r := {"hello": "world"} if {...}`,
`s(x) := 12 if {...}`. Not eligible: partial sets, or any rule whose value contains a variable
(`v := x if { x := true }`, `w := {"foo": x} if {...}`).

Two implications for this project:

- A boolean `allow` gate can exit early. A `violations` partial set **cannot** — by
  construction it must enumerate everything. That is a correct trade (you want all the
  reasons) but it means the deny-set shape is inherently the more expensive one, and its cost
  scales with the diff.
- Mixing values in one rule set defeats early exit. The docs' example: a rule set containing
  both `allow if ...` and `allow := false if ...` returns rules with different values from the
  index, so evaluation cannot stop at the first match. Keep `allow` uniformly `true`-valued
  and put the negative case in `default`.

### Comprehension indexing

Group-by in Rego would be O(n²) because there is no mutation. OPA mitigates this: "OPA may
compute and memoize the entire collection produced by comprehensions at once. This ensures
that runtime complexity is O(n)"
(<https://www.openpolicyagent.org/docs/policy-performance#comprehension-indexing>).

The five conditions for a comprehension to be indexed, verbatim:

> 1. The comprehension appears in an assignment or unification statement.
> 2. The expression containing the comprehension does not include a `with` statement.
> 3. The expression containing the comprehension is not negated.
> 4. The comprehension body is safe when considered independent of the outer query.
> 5. The comprehension body closes over at least one variable in the outer query and none of
>    these variables appear as outputs in references or `walk()` calls or inside nested
>    comprehensions.

Condition 2 is a trap for testing: **a `with` statement in the expression disables
comprehension indexing.** A group-by rule that measures 2.7 ms in production can measure far
worse under `opa test` where every assertion carries `with input as ...`, and conversely a
benchmark taken with `with` understates production speed. Benchmark with `opa bench -i`, not
through a test.

Even indexed, the group-by measured 2.73 ms against 12 µs for the pre-computed equivalent.
Comprehension indexing saves you from catastrophe, not from the cost.

### How to avoid accidental quadratic behaviour over large file lists

The two shapes that cost 200–350 ms in my measurements:

```rego
# BAD: cross-product. 346 ms / 69 MB on 500 paths.
pairs contains [p, q] if {
    some p in input.paths
    some q in input.paths
    p != q
    dirname(p) == dirname(q)
}

# BAD: nested scan — one inner iteration per outer path. 221 ms on 500 paths.
needs_test contains p if {
    some p in input.paths
    endswith(p, ".go")
    not has_test(p)
}
has_test(p) if {
    some q in input.paths          # <-- rescans the whole list per p
    q == replace(p, ".go", "_test.go")
}
```

The fix for the second is a set built once, which turns each inner scan into a hash probe —
1.24 ms, a 178× improvement, verified:

```rego
# GOOD
path_set := {p | some p in input.paths}

needs_test contains p if {
    some p in input.paths
    endswith(p, ".go")
    not endswith(p, "_test.go")
    not replace(p, ".go", "_test.go") in path_set
}
```

The rules that follow from this:

1. **Never nest two iterations over the same collection.** If you need pairs, you need a
   different input shape.
2. **Build a set once, at package level, and probe it.** `{p | some p in input.paths}` is
   computed once per evaluation and reused by every rule that references it.
3. **Keep joins against `data` bounded.** The ownership-table join in §3 is (paths × patterns).
   Fine at 20 patterns; not fine at 2,000. If the table grows, index it by directory in `data`
   so the lookup is a reference rather than a scan.
4. **Push reshaping into the host.** This is rule 1 restated: a cross-product you cannot avoid
   in Rego is usually a field you should have been given.
5. **Bound the input.** A 10,000-file PR will find whatever superlinearity your policy has.
   The host should cap the diff size it will evaluate and treat an over-cap diff as its own
   routing decision, not hand it to the engine.

### Tools

```console
$ opa eval -d policy.rego -i input.json --profile --profile-limit 5 'data.ci.guard.decision'
```

Verified output includes per-line `Time`, `Num Eval`, `Num Redo`, `Num Gen Expr`, and
`Location` — so the profiler points at a source line, which is what you need. `Num Redo`
columns in the thousands on a 500-path input is the signature of an unintended nested
iteration.

```console
$ opa bench -d policy.rego -i input.json 'data.ci.guard.decision'
$ opa test --bench ./policy/        # benchmark the unit tests
```

`opa test --bench` is the one to wire into CI as a regression gate, since it benchmarks what
the tests already cover. `--count N` plus `benchstat` gives comparable numbers across commits.

---

## 7. Failure modes and guardrails

### The three stages, and what fails in each

OPA's errors guide divides evaluation into **parsing**, **compilation**, and **evaluation**:
"Errors reported in any of these stages will stop the evaluation process"
(<https://www.openpolicyagent.org/docs/errors>). Knowing which stage an error belongs to tells
you when it can hurt you.

| Stage | Category | Example | When it bites |
|---|---|---|---|
| parsing | `rego_parse_error` | non-terminated string; `if` missing before a body | bundle load / CI |
| compilation | `rego_type_error` | `conflicting rules found`, arity mismatch, `undefined ref` (with `--schema`) | bundle load / CI |
| compilation | `rego_unsafe_var_error` | `var x is unsafe` | bundle load / CI |
| compilation | `rego_recursion_error` | `rule data.t.p is recursive: p -> q -> p` | bundle load / CI |
| compilation | `rego_compile_error` | `assigned var x unused` (strict mode only) | CI, if you enable strict |
| **evaluation** | `eval_conflict_error` | `complete rules must not produce multiple outputs` | **production, on one specific PR** |
| **evaluation** | `eval_builtin_error` | `div: divide by zero` | **production**, only under strict built-in errors |

Everything in the first group is caught by `opa check` before a bundle ever loads. The
`eval_*` errors are the ones that reach production, and `eval_conflict_error` is the one to
design against — verified in §1, it is input-dependent and invisible to a test suite that
never constructs the colliding input.

Verified recursion detection:

```console
$ opa check r.rego
2 errors occurred:
r.rego:2: rego_recursion_error: rule data.t.p is recursive: data.t.p -> data.t.q -> data.t.p
```

### Built-in errors are undefined by default — which is a fail-open default

> By default, built-in function calls that encounter runtime errors evaluate to undefined
> (which can usually be treated as `false`) and do not halt policy evaluation.

— <https://www.openpolicyagent.org/docs/policy-language#errors>

"Can usually be treated as false" is doing a lot of work in that sentence. For a `deny` rule,
undefined means **the denial does not fire**. A malformed value that makes `glob.match` return
undefined silently removes a control.

Strict built-in errors turns these into halting errors, and the docs give the full matrix:

| API | Flag |
|---|---|
| `POST v1/data` / `GET v1/data` (HTTP) | `strict-builtin-errors` query parameter |
| `opa eval` (CLI) | `--strict-builtin-errors` |
| `opa run` (REPL) | `> strict-builtin-errors` |
| `rego` Go module | `rego.StrictBuiltinErrors(true)` |
| Wasm | Not available |

For a security control plane, strict built-in errors is the right setting: it converts a
silent fail-open into a loud error the host can fail closed on. The cost is that a malformed
value anywhere in a large policy set aborts the whole evaluation — which argues again for
validating input at the host boundary so policy never sees a malformed value.

(One documented interaction: a prefix/suffix statement whose search operand is not a string is
*excluded by the index*, so the type error is never raised — visible only under strict
built-in errors. See the note at
<https://www.openpolicyagent.org/docs/policy-performance#prefix-and-suffix-statements>.)

### Distinguishing "policy said no" from "policy is broken"

The language gives you three outcomes and the host must treat them differently:

1. **A value** — policy decided. Honour it.
2. **Undefined** — policy did not decide. **Not a denial.**
3. **An error** — policy is broken, or input is malformed.

The only way to be sure which you have is for the policy to make its answer **total**: always
produce a value, and make the value self-describing. A bare boolean cannot distinguish "allow"
from "nothing matched", because `undefined` and `false` both look like absence to a caller
that defaults.

```rego
package ci.guard

default decision := {"effect": "block", "reasons": ["policy produced no decision"]}

decision := {"effect": effect, "reasons": sort(reasons)} if {
    input_is_wellformed
    reasons := {r | some r in violations}
    effect := effect_for(reasons)
}

input_is_wellformed if is_array(input.paths)

effect_for(reasons) := "block" if count(reasons) > 0
effect_for(reasons) := "allow" if count(reasons) == 0
```

The host then needs only three branches: an error from the engine, `undefined` (which this
policy makes impossible but the host must still handle), or an object with `effect` and
`reasons`.

### `default` does not give you fail-closed — I proved this on my own example

This is the most important caveat in this document, and I found it by running my own test
suite rather than by reading.

The policy above originally lacked the `input_is_wellformed` line. With `input = {}`:

- `violations` is a partial set; with no input, nothing contributes; it is `{}` — **not
  undefined, just empty.**
- `effect_for({})` therefore returns `"allow"`.
- `decision` is therefore **defined**, as `{"effect": "allow", "reasons": []}`.
- `default decision` never fires, because defaults apply only "when all the rules sharing the
  same name are undefined".

```console
$ opa test v/ -v
data.ci.guard_test.test_empty_input_fails_closed: FAIL
  ...
  policy.rego:12 | | | | Fail gt(__local74__, 0)
  policy_test.rego:29 | | Fail d.effect = "block"
PASS: 5/6
FAIL: 1/6
```

A malformed webhook produced a clean **allow**. The `default ... := block` was present the
whole time and did nothing.

The lesson generalises: **a `default` rule protects against undefined, and a deny-set
architecture is never undefined.** `count(deny) == 0` is structurally fail-open — an empty
deny set is indistinguishable from "no rule managed to run". The fix is to require a positive
assertion that the input was well-formed before any allow can be produced, and to test that
assertion explicitly. With `input_is_wellformed` added, 6/6 pass.

### Fail closed on security, fail open elsewhere

The project needs both behaviours, which means the *decision document* must say which kind of
decision it is. The language supports this cleanly, because separate rules produce separate
documents:

```rego
package ci.guard

# Security controls: the host must fail closed on error or undefined.
security_violations contains msg if { ... }

# Advisory routing: the host may proceed on error or undefined.
routing_suggestions contains team if { ... }

decision := {
    "effect": effect,
    "security_reasons": sort([m | some m in security_violations]),
    "routing": sort([t | some t in routing_suggestions]),
} if {
    input_is_wellformed
    ...
}
```

Keep the two in **separate packages** and evaluate them as **separate queries**, so that an
error in routing cannot take down the security decision and a host that fails closed on one
query can fail open on the other. Combining them into one document means one
`eval_conflict_error` in a routing rule blocks every PR.

### FAQ guidance on policy style and security posture

The FAQ compares three authoring styles
(<https://www.openpolicyagent.org/docs/faq#how-do-i-write-policies-securely>), and the
analysis maps onto this project well:

- **Default allow** (`default deny := false`; write `deny` rules): good for operations,
  "less appropriate [for security] because it allows unknown bad actions to occur".
- **Default deny** (`default allow := false`; write `allow` rules): "the only requests that
  are accepted are known to be safe", but you must enumerate everything legitimate up front.
- **Default allow with deny override**: `authz if { allow; not deny }`. "Any deny statements
  that you forget lead to security problems; any allow statements you forget lead to
  operational problems."

A CI control plane is the default-allow case — you cannot enumerate every safe PR — which
means the project is structurally in the weaker security posture and must compensate with the
guardrails below. Be explicit about that with users rather than implying the engine is
protecting them.

Note also the FAQ's conflict-resolution point: "Neither `allow` nor `deny` are keywords in
Rego so if you want to treat them as contradictory, you control which one takes precedence
explicitly" (<https://www.openpolicyagent.org/docs/faq#conflict-resolution>). The entry-point
rule the host queries is the only place precedence exists. Name it, document it, and never let
end-user policy define it.

### Guardrails against badly written policy

Ordered by value per unit of effort, all verified except where noted.

**1. `opa check --schema schemas/` — the single highest-value guardrail.** Converts a silent
fail-open typo into a compile error with a suggestion (§4). Nothing else catches this class.

**2. `import future.keywords.not` everywhere `not` appears.** Changes unknown-input cases from
fail-open to fail-closed (§2, verified table). Lint for its presence.

**3. `opa check --strict`.** Two additional checks, per
<https://www.openpolicyagent.org/docs/policy-language#strict-mode>: unused local assignments
and unused imports. Verified:

```console
$ opa check --strict s.rego
1 error occurred: s.rego:2: rego_compile_error: assigned var x unused
```

An unused assignment is usually a rule that silently does less than its author thinks.

**4. Regal, the Rego linter.** `github.com/open-policy-agent/regal` — in the OPA GitHub
organisation, v0.43.0 at time of writing, documented under
<https://www.openpolicyagent.org/projects/regal>. The OPA docs themselves link to Regal rules
for the exact bug classes in §1, notably
<https://www.openpolicyagent.org/projects/regal/rules/bugs/not-equals-in-loop> for the
existential-`!=` mistake. It also ships a language server, so authors get the feedback in
their editor. I verified Regal's repository, maintainer org, and release version, but **did not
install or run Regal**, so its specific rule behaviour here is unverified.

**5. `opa fmt --fail` in CI.** Verified: exits `0` on formatted input, non-zero when
reformatting is needed. One canonical layout removes a whole category of review noise. Useful
flags: `-d/--diff`, `-l/--list`, `-w/--write`, `--fail`, `--check-result` (default true).

**6. `opa test --fail-on-empty --coverage --threshold N --var-values`.** Verified.

**7. Constrain what policy can do at all.** `--capabilities` (present on `opa test`, `opa
check`, `opa fmt`, `opa eval`) restricts the available built-in set. Removing `http.send`,
`net.lookup_ip_addr`, and `opa.runtime` from end-user policy eliminates network calls,
nondeterminism, and runtime-secret access in one move, and keeps the latency budget honest. I
verified the flag exists on these commands and that the capability file format lists built-ins
by name; I **did not verify** end-to-end that a restricted capabilities file rejects a policy
calling an excluded built-in.

**8. Give authors a template, not a blank file.** Most of §1's bug classes are prevented by
starting from a skeleton with `default decision`, `input_is_wellformed`, the
`future.keywords.not` import, and the malformed-input tests already written. Authoring
ergonomics here is mostly a question of what the first file looks like.

### Metadata annotations for operational context

`# METADATA` blocks attach structured YAML to packages and rules, readable from policy itself
via `rego.metadata.rule()` and `rego.metadata.chain()`
(<https://www.openpolicyagent.org/docs/policy-language#metadata>):

```rego
# METADATA
# title: Auth middleware requires human sign-off
# description: Any change under internal/auth must be reviewed by a human.
# related_resources:
# - https://docs.example.com/policy/AUTH-1
# custom:
#   severity: high
#   owner: security
violations contains msg if { ... }
```

For this project that is the natural home for severity, owning team, and documentation links,
so a blocked PR can be told *why* and *who to ask*. `labels` (a flat string map) is a separate
annotation from `custom`. `opa inspect` reads annotations from a bundle, and the style guide
recommends annotations over free comments
(<https://www.openpolicyagent.org/docs/style-guide#use-metadata-annotations>). I verified the
annotation syntax and the `rego.metadata.*` built-ins exist in the v1.21.2 manifest; I did
**not** verify an end-to-end `rego.metadata.chain()` read.

---

## 8. Honest limitations

**Undefined is a design flaw you will manage forever.** Every section above kept returning to
it. A language where a missing field silently cancels a security control, and where the fix is
a stylistic convention (negate a helper) rather than a type error, puts the burden on every
author on every rule. `--schema` mitigates the typo case; nothing mitigates the genuinely
absent field except a host that never emits one. Budget for teaching this and for catching it
in review.

**Runtime conflicts in complete rules are input-dependent landmines.** `eval_conflict_error`
compiles clean, passes tests, and fires on the one PR that satisfies two bodies. In a system
where end users author policy and bodies are added independently, this is a matter of when. The
only real defences are architectural: prefer partial sets, and reduce in the host.

**Declarative without a type system is harder than it looks.** The gradual type system only
engages when you supply schemas, and the schema checker ignores `enum`, `oneOf`, `not`,
`additionalProperties`, `patternProperties`, `contains`, and `if`/`then`/`else`
(<https://www.openpolicyagent.org/docs/policy-language#limitations>). You get reference and
coarse type checking, not validation.

**Rego is a bad data-transformation language.** No mutation means every regrouping is a
comprehension, and the measurements in §6 show what that costs: 2.73 ms for one group-by
versus 12 µs for the same answer pre-computed. Teams reach for data wrangling in Rego because
the language *lets* them, and the result is slow policy that is hard to read. The architectural
answer is to put reshaping in the host — which is a limitation of Rego dressed up as a design
principle.

**The performance cliff is invisible at authoring time.** 7 µs to 346 ms is four orders of
magnitude, and nothing in the language, the linter, or the test runner warns you as you cross
it. A policy tested against a three-file fixture gives no signal about a 500-file PR. The
project needs `opa test --bench` with large fixtures as a CI gate, because authors will not
find this themselves.

**The learning curve is front-loaded and genuinely steep.** The three things an experienced
programmer must unlearn — order doesn't matter, `!=` inside an iteration does not mean "none
match", undefined is not false — all look like working code when you get them wrong. This is
the most commonly reported frustration with Rego adoption, and it is a real cost, not a
documentation gap.

**The language is still moving, and the docs run ahead of the releases.** `import rego.v2` is
documented in the current `not` and `import` keyword references as though available; it is
listed under **Unreleased** in the CHANGELOG and is a parse error on 1.21.2 (verified). That
means authoring guidance has a version axis end users will not track. Pin a version, pin the
documentation you hand out, and lint for imports your deployed OPA does not support.

**Debugging is tracing, not stepping.** `print`, `--explain`, `--var-values`, and the profiler
are good tools, but there is no breakpoint and no way to inspect a binding mid-search. Working
out *why* a rule is undefined on one input is still an exercise in bisecting the body — which
is the practical argument for the style guide's advice to prefer many small named rules you can
query individually
(<https://www.openpolicyagent.org/docs/style-guide#consider-partial-helper-rules-over-comprehensions-in-rule-bodies>).

**Where Rego is a poor fit for this project specifically.** Anything needing to look at diff
*content* with structure — "did this change weaken a specific conditional", "is this IAM policy
semantically broader than before" — is parsing, and Rego has no parser. Regex over
`added_lines` works for a shallow signal and will produce false positives and negatives. If
semantic diff analysis becomes a requirement, it belongs in Go, with Rego deciding over the
parsed result. Equally, any policy needing to call out (fetching a file at a ref, checking a
ticket) wants `http.send`, which forfeits both the latency budget and determinism. Resolve
external facts in the host and put them in `input`.

**What Rego is genuinely good at here**, to be fair to it: declaring many independent
conditions over a known-shape JSON document, combining them without coordination between
authors, and answering in microseconds. That is precisely the job in §3. The failure modes
above are all failures of *using it for something else* — data transformation, content
parsing, external lookups — or of the undefined semantics. Keep the input document rich and
the policies thin and Rego is a good fit.

---

## Appendix: source index

| Topic | URL |
|---|---|
| Language reference | <https://www.openpolicyagent.org/docs/policy-language> |
| — Rules, incremental definitions | <https://www.openpolicyagent.org/docs/policy-language#rules> |
| — Negation | <https://www.openpolicyagent.org/docs/policy-language#negation> |
| — Universal quantification | <https://www.openpolicyagent.org/docs/policy-language#universal-quantification-for-all> |
| — `with` | <https://www.openpolicyagent.org/docs/policy-language#with-keyword> |
| — `default` | <https://www.openpolicyagent.org/docs/policy-language#default-keyword> |
| — Built-in errors | <https://www.openpolicyagent.org/docs/policy-language#errors> |
| — Metadata | <https://www.openpolicyagent.org/docs/policy-language#metadata> |
| — Schemas / type checking | <https://www.openpolicyagent.org/docs/policy-language#schema> |
| — Strict mode | <https://www.openpolicyagent.org/docs/policy-language#strict-mode> |
| Policy reference (grammar, operators) | <https://www.openpolicyagent.org/docs/policy-reference> |
| Built-ins: glob | <https://www.openpolicyagent.org/docs/policy-reference/builtins/glob> |
| Built-ins: strings | <https://www.openpolicyagent.org/docs/policy-reference/builtins/strings> |
| Built-ins: sets | <https://www.openpolicyagent.org/docs/policy-reference/builtins/sets> |
| Keyword: `not` (improved semantics) | <https://www.openpolicyagent.org/docs/policy-reference/keywords/not> |
| Keyword: `and` / `or` | <https://www.openpolicyagent.org/docs/policy-reference/keywords/logical> |
| Keyword: `import` (future keywords, rego.v1/v2) | <https://www.openpolicyagent.org/docs/policy-reference/keywords/import> |
| Policy testing | <https://www.openpolicyagent.org/docs/policy-testing> |
| Policy performance | <https://www.openpolicyagent.org/docs/policy-performance> |
| — Indexed statements | <https://www.openpolicyagent.org/docs/policy-performance#use-indexed-statements> |
| — Early exit | <https://www.openpolicyagent.org/docs/policy-performance#early-exit-in-rule-evaluation> |
| — Comprehension indexing | <https://www.openpolicyagent.org/docs/policy-performance#comprehension-indexing> |
| Style guide | <https://www.openpolicyagent.org/docs/style-guide> |
| Errors guide | <https://www.openpolicyagent.org/docs/errors> |
| FAQ | <https://www.openpolicyagent.org/docs/faq> |
| Cheatsheet | <https://www.openpolicyagent.org/docs/cheatsheet> |
| Upgrading to v1.0 | <https://www.openpolicyagent.org/docs/v0-upgrade> |
| Regal (linter + language server) | <https://www.openpolicyagent.org/projects/regal> |
| Capability manifest, v1.21.2 | <https://github.com/open-policy-agent/opa/blob/main/capabilities/v1.21.2.json> |
| CHANGELOG (feature version history) | <https://github.com/open-policy-agent/opa/blob/main/CHANGELOG.md> |
| `opa test` flag definitions (source) | <https://github.com/open-policy-agent/opa/blob/main/cmd/test.go> |
| Rego Playground | <https://play.openpolicyagent.org/> |
