# Rego — policy language and authoring

Knowledge base entry for the Open CI Control Plane. Read this before `rego-language.md`,
which holds the detail.

Companion entry: `opa-overview.md` / `opa-embedding-and-prior-art.md` — the engine that
evaluates these policies, how it is embedded, and how policy is distributed.

Pinned to **OPA v1.21.2** (released 2026-10-12).

## What Rego is

**Rego is Open Policy Agent's policy language** — the language the rules are written in. OPA
is the engine; Rego is what you write.

It is *declarative* and descends from **Datalog**, which is the single most important thing
to know about it, because it means Rego is not a scripting language with unfamiliar syntax.
**It is a query language.**

You do not write a procedure that runs top to bottom and returns a value. You declare **rules
that define documents**, and when OPA is asked for a value it *searches* for variable
bindings that satisfy those rules. The official framing is worth internalising verbatim:

> When evaluating rule bodies, OPA searches for variable bindings that make all of the
> expressions true. There may be multiple sets of bindings that make the rule body true.

```
rule body  ==  expression-1 AND expression-2 AND ... AND expression-N
rule       ==  rule-name IS value IF body
```

There are **no loops, no mutation, no early return**, and **statement order carries no
meaning**. An experienced programmer's instincts are actively misleading here, which is why
the undefined section is the longest one in the detail document.

## Where a policy lives

| Term | What it means |
|---|---|
| **Module** | One `.rego` file. |
| **Package** | A module's namespace, declared at the top. `package ci.classify` means its rules live at `data.ci.classify`. **This, not the file path, determines where a rule lives.** |
| **Complete rule** | One name, one value — `allow`, `max_files`, a decision object. Multiple definitions must agree or evaluation errors. |
| **Partial rule** | A set or object built from many matches — a `violations` set. Multiple definitions **union**, which is what lets many authors contribute without coordinating. |
| **`input`** | The request-scoped document being decided about. Here: a JSON representation of a PR diff. |
| **`data`** | The static document tree — reference facts, plus every rule any loaded module defines. |
| **Undefined** | A third outcome alongside true and false, meaning "no value was produced". **Not an error and not a denial.** The central hazard of this language. |

## The smallest complete example

```rego
# policy.rego
package ci.guard

# A partial set. Each matching clause contributes one element.
violations contains "auth middleware changed without a test" if {
    some path in input.paths
    startswith(path, "internal/auth/")
    count(test_files) == 0
}

# A helper rule, itself a partial set.
test_files contains path if {
    some path in input.paths
    endswith(path, "_test.go")
}

# A complete rule: one object, always produced.
decision := {"effect": effect, "reasons": sort([v | some v in violations])} if {
    is_array(input.paths)
    effect := effect_for(count(violations))
}

effect_for(n) := "block" if n > 0
effect_for(n) := "allow" if n == 0
```

```console
$ cat input.json
{"paths": ["internal/auth/middleware.go", "README.md"]}

$ opa eval -d policy.rego -i input.json 'data.ci.guard.decision'
{
  "effect": "block",
  "reasons": ["auth middleware changed without a test"]
}
```

Three things here recur everywhere else:

- `violations` is a **partial set**, so any number of independently-written clauses accumulate
  into it without coordinating. That is the property that lets many teams add rules to one
  policy.
- `decision` is a **complete rule** producing an object rather than a boolean, so the host
  receives the reasons alongside the verdict.
- `is_array(input.paths)` is a **well-formedness guard**. Without it, a malformed input
  produces an empty violation set and therefore a clean `allow`.

**Caveat on this example only:** it was composed from patterns verified individually in
`rego-language.md`, but the example as a whole was not itself executed. Every other code
sample in that document was run against the OPA 1.21.2 binary.

## The headline findings

Detail, measurements and code in `rego-language.md`.

1. **`default` does not give you fail-closed.** A deny-set architecture is never undefined, so
   `default decision := block` never fires. Proven on a live policy: malformed input produced
   a clean **allow**.
2. **A mistyped `input` path compiles clean, evaluates to undefined forever, and therefore
   fails open.** `opa check --schema` is the only thing that catches it, and it suggests the
   correct field name.
3. **Built-in runtime errors evaluate to undefined by default**, which silently deletes a
   `deny` rule.
4. **Evaluation latency spans 7 µs to 346 ms on the same question**, depending entirely on how
   the input document is shaped. Pre-computing in the host gave a 220× speedup.
5. **`import future.keywords.not` flips unknown-subject cases from fail-open to fail-closed.**
   Treat it as compulsory and lint for its presence.
6. **`eval_conflict_error` is an input-dependent landmine.** Complete rules that disagree
   compile clean, pass the whole test suite, and fail in production on the one PR that
   satisfies both bodies.
7. **A CI control plane is structurally default-allow-with-deny-overrides**, because you cannot
   enumerate every safe PR. Every forgotten deny is a hole. Be explicit with users rather than
   implying the engine protects them.
