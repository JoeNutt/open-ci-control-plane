# OPA — policy engine, Go embedding and distribution

Knowledge base entry for the Open CI Control Plane. Read this before
`opa-embedding-and-prior-art.md`, which holds the detail.

Companion entry: `rego-overview.md` / `rego-language.md` — the language the policies
this engine evaluates are written in.

All API details pinned to **OPA v1.21.2** (released 2026-10-12).

## What OPA is

**Open Policy Agent is a general-purpose policy engine.** Its purpose is to take the
question "is this allowed?" out of application code and put it somewhere else. You hand OPA
a JSON document describing something that is happening — a request, a deployment, a pull
request — and it evaluates rules you wrote against it and returns a decision, also as JSON.

**OPA never enforces anything. It only answers.** The calling service asks the question and
is wholly responsible for acting on the answer. That division is the single most important
thing to hold on to, because every fail-open hazard in these documents exists because a
service can ask a question, receive silence, and carry on.

It is a CNCF **graduated** project under Apache-2.0, and it is the engine behind Kubernetes
admission control (via Gatekeeper), Envoy authorization, and Terraform plan gating.

The rules themselves are written in **Rego**, OPA's declarative policy language. Rego has
its own entry; this one is about the engine, how it is embedded in a Go service, and how
policy reaches it.

## The model in one line

```
input  (the thing being decided about, JSON, per request)
  +
data   (static facts the policy can consult, JSON)
  +
policy (rules written in Rego)
  ↓
decision (JSON — whatever shape the rules produce)
```

## Vocabulary

| Term | What it means |
|---|---|
| **`input`** | The request-scoped document being decided about. Here: a JSON representation of a pull-request diff. Supplied fresh on every evaluation. |
| **`data`** | The static document tree — reference facts the policy can look up (ownership tables, path classifications), plus every rule any loaded module defines. Changes rarely. |
| **Query** | The path the host asks for, e.g. `data.ci.pr.decision`. The entry point into the policy. |
| **Decision** | Whatever that query evaluates to. Can be a boolean, but here should be an object carrying a verdict *and* its reasons. |
| **Undefined** | A third outcome alongside true and false, meaning "no value was produced". **Not an error and not a denial.** The central hazard of this whole technology. |
| **Bundle** | A gzipped tarball of policies and data, optionally signed, which is how policy is distributed to a running OPA. **The mechanism this project's thesis depends on.** |
| **Built-in** | A provided function — `glob.match`, `count`, `http.send`. There are 206 in v1.21.2, and which ones you permit is a security decision. |
| **Capabilities** | The declared set of built-ins a policy is allowed to call. The main containment tool for policy you did not write. |
| **Module / package / rule** | Language-level concepts. See the Rego entry. |

## Three ways OPA is deployed

1. **As a Go library, embedded in your own process.** No network hop, microsecond decisions.
   **This is what the control plane does**, and the detail document is about this mode.
2. **As a separate daemon or sidecar**, asked over HTTP. Language-agnostic, at the cost of a
   network round trip per decision.
3. **As a CLI** — `opa eval`, `opa test`, `opa bench`, `opa check`, `opa fmt`. How policy
   gets written, tested and benchmarked before it ever ships. Every measurement in these
   documents came from these.

## What this means for the control plane, concretely

1. GitHub sends a pull-request webhook.
2. The control plane turns it into a **diff document** — changed paths, file metadata, added
   lines, pre-computed classifications. That document is `input`, and designing it is the
   project's largest single design decision (covered in the Rego entry).
3. The control plane evaluates its **query** against the policy currently loaded, which
   arrived as a **signed bundle** from a repository the governed developers cannot write to.
4. The **decision** comes back: block, allow, or route to named reviewers, with reasons.
5. The control plane *enforces* it — a GitHub check run, a blocked merge, a dispatched
   pipeline.

## Why use OPA at all, rather than if-statements in Go

Worth being blunt about, because it is also the conclusion of the alternatives assessment.
Writing these checks directly in Go would be simpler, faster and completely type-safe. What
you would lose is the entire reason this project exists:

- **Policy changes without redeploying the service.**
- **Policy authored by people who do not commit to your codebase** — platform and security
  teams, per repository.
- **Policy as a signed, versioned artifact** that lives outside the repository it governs and
  cannot be edited by the developers, or the coding agents, it constrains.
- **Every decision logged against the policy revision that produced it**, so a blocked PR can
  be explained months later.

That list is "policy lifecycle", and it is **the one axis on which OPA beats every
alternative examined**. If those four things turn out not to be requirements, the
alternatives section explains why plain Go predicates would be the better choice.

## The headline findings

Detail, measurements and code in `opa-embedding-and-prior-art.md`.

1. **`opa/rego` is deprecated — use `opa/v1/rego`.** The `v1/` is a directory inside the
   module, not a major-version suffix.
2. **The `rego` package cannot satisfy the distribution or audit requirements.** Bundles,
   signature verification and decision logs are plugins reachable only through `opa/v1/sdk`,
   configured by a JSON document.
3. **OPA has no per-evaluation memory, allocation or step limit.** Established by enumerating
   all 37 evaluator options. The only bound is a context deadline. A legal, terminating
   tenant policy can OOM-kill the whole control plane. **The strongest single argument
   against OPA.**
4. **An empty `ResultSet` means undefined, not false.** The classic integration bug.
5. **Latency is ~36 µs median but ~336 µs at p99.9** — state the requirement as a p99 target.
6. **Bundle signing covers the file *set*, not just contents** — an attacker can neither add
   an unsigned file nor remove a signed one. This is what makes the thesis technical rather
   than procedural.
7. **Gatekeeper's three-way failure split**: unreachable → allow; engine errored → deny;
   self-exemption attempt → always deny.
