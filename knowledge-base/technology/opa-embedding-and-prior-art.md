# Embedding and Operating OPA Inside a Go Service

Reference material for a CI control plane: a Go service between GitHub and a CI runner that
evaluates declarative policy against the structure of a pull-request diff.

**Version pinned for all API details: OPA v1.21.2, released 2026-10-12.** Everything in the
Go API sections was read from the tagged source tree at
`https://github.com/open-policy-agent/opa/tree/v1.21.2`, not from memory or from
documentation prose. Where a claim comes only from the documentation site and not from source,
that is stated.

Scope note: this document deliberately does **not** cover the Rego language, authoring idioms,
or policy testing. It covers Go integration, operations, prior art, and alternatives.

---

## Read This First: Two Corrections to the Brief

Two premises in the task as given are out of date or incomplete, and both change the design.

**1. `github.com/open-policy-agent/opa/rego` is deprecated. The current path is
`github.com/open-policy-agent/opa/v1/rego`.**

The module path is `github.com/open-policy-agent/opa` — confirmed from
[`go.mod`](https://raw.githubusercontent.com/open-policy-agent/opa/v1.21.2/go.mod), which reads
`module github.com/open-policy-agent/opa` with `go 1.26.0`. Note it is *not* `/v2`; the major
version is 1, and the `v1/` in the import path is a **directory inside the module**, not a Go
module major-version suffix. This is an unusual layout and a common source of confusion.

pkg.go.dev for the old path states the package is deprecated and "intended for older projects
transitioning from OPA v0.x", recommending `github.com/open-policy-agent/opa/v1/rego` instead;
the old path is a thin wrapper around the new one
([pkg.go.dev/github.com/open-policy-agent/opa/rego](https://pkg.go.dev/github.com/open-policy-agent/opa/rego)).
The practical difference is the default Rego language version: the `v1` packages default to
Rego v1 (where `if` and `contains` are mandatory keywords and `import rego.v1` is unnecessary),
while the legacy path defaults to v0 semantics. **Use `opa/v1/rego` for new code.**

**2. The `rego` package alone will not satisfy requirements 3 and 5. Bundles and decision logs
live in the `sdk` package.**

This is the single most important architectural fact in this document. OPA's own integration
guide splits the Go story in two:

> Use the low-level `github.com/open-policy-agent/opa/v1/rego` package to embed OPA as a library
> inside services written in Go, **when only policy evaluation — and no other capabilities of OPA,
> like the management features — are desired.** If you're unsure which one to use, the SDK is
> probably the better option.
>
> — [Integrating OPA](https://www.openpolicyagent.org/docs/integration)

And, in the comparison table at the end of that same page, for the "Go Lib" column under
Operations: *"All of the management functionality (bundles, decision logs, etc.) must be either
enabled or implemented."*

Bundle polling, bundle signature verification, delta bundles, status reporting and decision
logging are **plugins managed by `plugins.Manager`**, which the `sdk` package constructs for you
from a JSON/YAML config document. They are not options on `rego.New`. The project's core thesis
(policy distributed from a central, developer-inaccessible source, with integrity and provenance)
and its audit requirement are therefore *both* arguments for `github.com/open-policy-agent/opa/v1/sdk`
over `opa/v1/rego`.

The recommendation is laid out in "Which Package Should This Project Use?" below, after both
APIs are described.

---

## 1. The Go Embedding API: `opa/v1/rego`

### 1.1 The shape of the API

Three types matter.

`rego.Rego` is a builder. It is constructed by `rego.New`, which takes a variadic list of
functional options:

```go
func New(options ...func(r *Rego)) *Rego
```

Note the option type: it is the bare `func(r *Rego)`, not a named `Option` type. Nearly every
configuration function in the package returns `func(r *Rego)`. (There are separate named types
`EvalOption`, `PrepareOption` and `CompileOption` for the other phases.)
Source: [`v1/rego/rego.go#L1440`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego.go).

`rego.PreparedEvalQuery` is the compiled, reusable artifact:

```go
func (r *Rego) PrepareForEval(ctx context.Context, opts ...PrepareOption) (PreparedEvalQuery, error)
func (pq PreparedEvalQuery) Eval(ctx context.Context, options ...EvalOption) (ResultSet, error)
```

**`PrepareForEval` returns a value, not a pointer**, and `Eval` is declared on the value receiver
`pq PreparedEvalQuery`. This is worth noticing because it is the structural reason the thing is
concurrency-safe: evaluation does not mutate the prepared query.
Source: [`v1/rego/rego.go#L1812` and `#L571`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego.go).

`rego.ResultSet` is the output. Its full definition is only ~100 lines and worth knowing exactly
([`v1/rego/resultset.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/resultset.go)):

```go
type ResultSet []Result

type Result struct {
    Expressions []*ExpressionValue `json:"expressions"`
    Bindings    Vars               `json:"bindings,omitempty"`
}

type Vars map[string]any

type ExpressionValue struct {
    Value    any       `json:"value"`
    Text     string    `json:"text"`
    Location *Location `json:"location"`
}
```

The semantics that matter:

- **An empty `ResultSet` means the query was *undefined*, not false.** This is a distinct third
  state from `true` and `false`, and conflating it with `false` is the classic OPA integration
  bug. For this project it is the difference between "policy says this PR is fine" and "no policy
  rule matched this PR at all" — which, on a security decision, must fail closed, and on a
  test-selection decision must fail open. Handle it explicitly.
- Values in `Expressions[].Value` and `Bindings` are plain Go types: `bool`, `string`,
  `json.Number`, `map[string]any`, `[]any`. There is no struct unmarshalling; if you want typed
  output, round-trip through `encoding/json` yourself.
- A query that produces multiple documents yields multiple `Result` elements, one per binding
  combination.

Two helpers on `ResultSet`, both verified from source:

```go
func (rs ResultSet) Allowed() bool
func ResultValue[T any](rs ResultSet) (T, bool)
```

`Allowed()` returns true only if the result set has exactly one element, that element has exactly
one expression, that expression's value is `true`, and **there are no bindings**. The source
comment explains the last condition: it would be a trap to return `true` for a query like
`data.authz.allow = x`, whose result set always has a single element with value `true` while the
actual decision hides in the binding `x`. The generic `ResultValue[T]` is the same contract for
any type `T`, and is the right tool when your decision is a struct or a list of violations rather
than a boolean. `Allowed()` is literally implemented as `ResultValue[bool](rs)`.

### 1.2 Compile once, evaluate many

This is the whole performance story in one sentence: `rego.New(...)` plus `PrepareForEval`
does parsing, compilation, type checking and query planning; `Eval` does only evaluation.

The documentation is explicit on both halves:

> Preparing queries in advance avoids parsing and compiling the policies on each query and
> improves performance considerably. **Prepared queries are safe to share across multiple Go
> routines.**
>
> — [Integrating OPA](https://www.openpolicyagent.org/docs/integration)

So the correct architecture is: **prepare once per policy version, store the
`PreparedEvalQuery` behind an atomic pointer, and have every webhook handler goroutine call
`Eval` on it concurrently.** When a new policy bundle arrives, prepare a new query and swap the
pointer. Never call `rego.New` on the request path.

The magnitude of the difference is not documented as a single number, but OPA's own benchmark
suite is structured precisely to expose it — [`v1/rego/rego_bench_test.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego_bench_test.go)
contains the paired benchmarks `BenchmarkAciTestBuildAndEval` and `BenchmarkAciTestOnlyEval`,
whose names describe the two paths. Independent of absolute figures, compilation involves
parsing Rego text, building and type-checking an AST, and running the rule indexer; evaluation
walks an already-built plan. Treating these as the same cost is how a sub-millisecond budget
becomes a tens-of-milliseconds budget.

One caveat on `Eval`: by default, each call spawns **a short-lived goroutine** to translate
`ctx` cancellation into OPA's internal `topdown.Cancel`. From
[`v1/rego/rego.go#L2384`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego.go):

```go
// Cancel query if context is cancelled or deadline is reached.
if ectx.externalCancel == nil {
    // Create a one-off goroutine to handle cancellation for this query.
    c := topdown.NewCancel()
    q = q.WithCancel(c)
    exit := make(chan struct{})
    defer close(exit)
    go waitForDone(ctx, exit, func() {
        c.Cancel()
    })
} else {
    // Query cancellation is being handled elsewhere.
    q = q.WithCancel(ectx.externalCancel)
}
```

At this project's decision rate (webhooks, not a service mesh) that goroutine is irrelevant. If
it ever matters, `rego.EvalExternalCancel(topdown.Cancel)` lets you supply cancellation yourself
and skip it.

### 1.3 A minimal end-to-end evaluation

Every identifier below was read from the v1.21.2 source tree. No Go toolchain was available in
this environment, so **this code has been verified identifier-by-identifier against source but
has not been compiled.** That caveat is real; treat the first build as a check.

```go
package prpolicy

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
)

// Decision is the policy verdict for one pull request.
type Decision struct {
	Allow            bool     `json:"allow"`
	RequiresReview   bool     `json:"requires_review"`
	Reasons          []string `json:"reasons"`
}

// Engine holds a compiled policy and evaluates pull requests against it.
// The zero value is not usable; construct with NewEngine.
type Engine struct {
	query atomic.Pointer[rego.PreparedEvalQuery]
}

// NewEngine compiles policySource and returns an Engine ready to evaluate.
func NewEngine(ctx context.Context, policySource string) (*Engine, error) {
	engine := &Engine{}
	if err := engine.Reload(ctx, policySource); err != nil {
		return nil, err
	}
	return engine, nil
}

// Reload compiles policySource and atomically replaces the active policy.
// Evaluations already in flight continue against the previous policy.
func (e *Engine) Reload(ctx context.Context, policySource string) error {
	prepared, err := rego.New(
		// Bind the whole decision document to x, not just a boolean, so the
		// policy can return reasons alongside the verdict.
		rego.Query("x = data.ci.pr.decision"),
		rego.Module("pr.rego", policySource),
		rego.Strict(true),
		rego.StrictBuiltinErrors(true),
		rego.Capabilities(restrictedCapabilities()),
	).PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("compile policy: %w", err)
	}
	e.query.Store(&prepared)
	return nil
}

// Evaluate returns the policy decision for one pull request diff.
func (e *Engine) Evaluate(ctx context.Context, input any) (Decision, error) {
	prepared := e.query.Load()
	if prepared == nil {
		return Decision{}, errors.New("no policy loaded")
	}

	resultSet, err := prepared.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return Decision{}, classifyEvalError(err)
	}

	// An empty result set means the query was UNDEFINED. That is not "false".
	if len(resultSet) == 0 {
		return Decision{}, ErrPolicyUndefined
	}

	raw, ok := resultSet[0].Bindings["x"].(map[string]any)
	if !ok {
		return Decision{}, fmt.Errorf("policy returned %T, want object", resultSet[0].Bindings["x"])
	}
	return decisionFromObject(raw)
}
```

The three things to carry away from that: the `atomic.Pointer[rego.PreparedEvalQuery]` swap, the
explicit `len(resultSet) == 0` undefined branch, and the fact that the query binds a whole
decision object rather than a bare boolean — because a PR-gating service needs the *reason*, not
just the verdict, and a second evaluation to fetch reasons doubles the cost for no benefit.

Reading the decision out of the `map[string]any` is mechanical and omitted. The cleaner
alternative, if the decision document is a fixed shape, is to use the generic helper and
round-trip through JSON:

```go
decisionDoc, ok := rego.ResultValue[map[string]any](resultSet)
```

Note `ResultValue` requires **no bindings**, so it works with `rego.Query("data.ci.pr.decision")`
(no `x =`) rather than the bound form above. Pick one style and be consistent.

### 1.4 Options on `rego.New` worth knowing

All verified present in [`v1/rego/rego.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego.go) at v1.21.2.

| Option | Signature | Why this project cares |
|---|---|---|
| `rego.Query` | `func Query(q string) func(r *Rego)` | The decision path to evaluate. |
| `rego.Module` | `func Module(filename, input string) func(r *Rego)` | Policy from a string. Filename is used in error locations — pass something meaningful. |
| `rego.ParsedBundle` | `func ParsedBundle(name string, b *bundle.Bundle) func(r *Rego)` | Load an already-parsed bundle. The bridge between the bundle format and the low-level API. |
| `rego.LoadBundle` | `func LoadBundle(path string) func(r *Rego)` | Load a bundle tarball from a filesystem path. |
| `rego.Store` | `func Store(s storage.Store) func(r *Rego)` | Supply the `data` document from a store rather than inline. |
| `rego.Capabilities` | `func Capabilities(c *ast.Capabilities) func(r *Rego)` | **The main untrusted-policy guardrail.** See §4. |
| `rego.StrictBuiltinErrors` | `func StrictBuiltinErrors(yes bool) func(r *Rego)` | Turns built-in errors from silently-undefined into fatal. See §4. |
| `rego.Strict` | `func Strict(yes bool) func(r *Rego)` | Compiler strict mode — rejects sloppy policy at compile time. |
| `rego.UnsafeBuiltins` | `func UnsafeBuiltins(unsafeBuiltins map[string]struct{}) func(r *Rego)` | Denylist of built-ins. Weaker than `Capabilities`; see §4. |
| `rego.EnablePrintStatements` | `func EnablePrintStatements(yes bool) func(r *Rego)` | Without this, `print()` calls are **erased** from the policy at compile time. |
| `rego.PrintHook` | `func PrintHook(h print.Hook) func(r *Rego)` | Where `print()` output goes. Useful for giving policy authors a debugging channel. |
| `rego.SetRegoVersion` | `func SetRegoVersion(version ast.RegoVersion) func(r *Rego)` | Pin `ast.RegoV0` or `ast.RegoV1` explicitly rather than relying on the import path's default. |
| `rego.SkipBundleVerification` | `func SkipBundleVerification(yes bool) func(r *Rego)` | Should remain **false** for this project. Named here so it is recognisable as a thing never to set. |
| `rego.Metrics` / `rego.Instrument` | `func Metrics(m metrics.Metrics) func(r *Rego)` | Per-phase timing — how you find out whether you are actually sub-millisecond. |

And on `Eval`:

| Option | Signature |
|---|---|
| `rego.EvalInput` | `func EvalInput(input any) EvalOption` |
| `rego.EvalParsedInput` | `func EvalParsedInput(ast.Value) EvalOption` |
| `rego.EvalMetrics` | `func EvalMetrics(metrics.Metrics) EvalOption` |
| `rego.EvalEarlyExit` | `func EvalEarlyExit(enabled bool) EvalOption` |
| `rego.EvalExternalCancel` | `func EvalExternalCancel(ec topdown.Cancel) EvalOption` |

`EvalParsedInput` deserves a note: `EvalInput(any)` has to convert your Go value into OPA's
internal `ast.Value` on every call. If the same input is evaluated against several prepared
queries — plausible here, if security policy and test-selection policy are separate bundles —
convert once with `ast.InterfaceToValue` and pass `EvalParsedInput` to each. (`ast.InterfaceToValue`
is used this way in OPA's own benchmarks.)

### 1.5 The `sdk` package

Verified from [pkg.go.dev/github.com/open-policy-agent/opa/v1/sdk](https://pkg.go.dev/github.com/open-policy-agent/opa/v1/sdk):

```go
func New(ctx context.Context, nopts Options) (*OPA, error)

type Options struct {
    Config        io.Reader
    Logger        logging.Logger
    ConsoleLogger logging.Logger
    Ready         chan struct{}
    Plugins       map[string]plugins.Factory
    ID            string
    Store         storage.Store
    Hooks         hooks.Hooks
    V0Compatible  bool
    V1Compatible  bool
    RegoVersion   ast.RegoVersion
    ManagerOpts   []func(manager *plugins.Manager)
}

func (opa *OPA) Decision(ctx context.Context, options DecisionOptions) (*DecisionResult, error)
func (opa *OPA) Partial(ctx context.Context, options PartialOptions) (*PartialResult, error)
func (opa *OPA) Configure(ctx context.Context, opts ConfigOptions) error
func (opa *OPA) Stop(ctx context.Context)

type DecisionOptions struct {
    Now                 time.Time
    Path                string
    Input               any
    NDBCache            any
    StrictBuiltinErrors bool
    Tracer              topdown.QueryTracer
    Metrics             metrics.Metrics
    Profiler            topdown.QueryTracer
    Instrument          bool
    DecisionID          string
    HTTPRoundTripper    topdown.CustomizeRoundTripper
}

type DecisionResult struct {
    ID         string
    Result     any
    Provenance types.ProvenanceV1
}

func IsUndefinedErr(err error) bool
const UndefinedErr = "opa_undefined_error"
```

The critical detail is `Config io.Reader`: **the SDK is configured by the same JSON/YAML
configuration document that `opa run -c` consumes.** Everything in §3 (bundles, signing,
polling) and §5 (decision logs) is reached by putting the corresponding keys in that document.
There is no Go-struct API for them; you build the config as bytes.

Note also that `DecisionResult.Result` is `any` and there is no `Bindings` concept — the SDK
evaluates a *named decision* at `Path` and hands back its value. The undefined case is surfaced
as an **error** (`sdk.IsUndefinedErr`), not an empty result set. That is a cleaner contract for
this project than the `rego` package's `len(rs) == 0`, and it is one more point in the SDK's
favour.

The verbatim example from OPA's integration guide, which also demonstrates `sdktest.NewServer`
and `sdktest.MockBundle` for testing a bundle-fed service without standing up a real bundle
server:

```go
package main

import (
    "bytes"
    "context"
    "fmt"

    "github.com/open-policy-agent/opa/v1/sdk"
    sdktest "github.com/open-policy-agent/opa/v1/sdk/test"
)

func main() {
    ctx := context.Background()

    // create a mock HTTP bundle server
    server, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", map[string]string{
        "example.rego": `
                package authz

                default allow := false

                allow if input.open == "sesame"
            `,
    }))
    if err != nil {
        // handle error.
    }

    defer server.Stop()

    config := []byte(fmt.Sprintf(`{
        "services": {
            "test": {
                "url": %q
            }
        },
        "bundles": {
            "test": {
                "resource": "/bundles/bundle.tar.gz"
            }
        },
        "decision_logs": {
            "console": true
        }
    }`, server.URL()))

    opa, err := sdk.New(ctx, sdk.Options{
        ID:     "opa-test-1",
        Config: bytes.NewReader(config),
    })
    if err != nil {
        // handle error.
    }

    defer opa.Stop(ctx)

    if result, err := opa.Decision(ctx, sdk.DecisionOptions{Path: "/authz/allow", Input: map[string]any{"open": "sesame"}}); err != nil {
        // handle error.
    } else if decision, ok := result.Result.(bool); !ok || !decision {
        // handle error.
    }
}
```

Source: [Integrating OPA → Integrating with the Go SDK](https://www.openpolicyagent.org/docs/integration).
The docs add that setting `ID` is optional but recommended, because otherwise a fresh random ID
is generated on every process restart, which makes a decision log impossible to follow over time.
For a multi-tenant audit trail, set it deliberately.

Two further SDK capabilities relevant here, both from the same page:

**Filesystem bundles with no server.** The SDK can load a bundle from a `file://` resource with no
`services` block at all, which is how you'd ship a default policy inside the container or load
from a mounted volume:

```go
config := []byte(fmt.Sprintf(`{
    "bundles": {
        "test": {
            "resource": "file://%s"
        }
    },
    "decision_logs": {
        "console": true
    }
}`, bundleFilePath))
```

**Manually triggered bundle reloads.** Set `bundles[_].trigger` to `manual` to disable periodic
polling, then drive reloads from an external event — a webhook from the policy repository, say,
which is a much better fit than polling for a system whose policy changes rarely:

```go
options := sdk.Options{
    Config: bytes.NewReader(config),
    Logger: logger,
    Ready:  make(chan struct{}), // <-- needed or else sdk.New will block
}

opa, err := sdk.New(ctx, options)
if err != nil {
    log.Fatal(err)
}
defer opa.Stop(ctx)

bundle, ok := opa.Plugin("bundle").(*bundle.Plugin)
if !ok {
    log.Fatal("bundle plugin not found")
}

// Monitor the status of the bundle download and activation by registering a listener function
bundle.Register("test", func(status bundle.Status) {
    // handle the status
})

err = bundle.Trigger(ctx)
if err != nil {
    log.Fatal(err)
}
```

Note the `Ready: make(chan struct{})` comment in OPA's own example — *"needed or else sdk.New will
block"*. `sdk.New` waits for plugins to reach ready state unless you give it a channel to signal
on. That is a real startup-ordering trap. Also note `opa.Plugin(name string)` returns `any`
requiring a type assertion, and `bundle.Status` is delivered to a registered listener — this is
the hook for exposing "which policy revision am I enforcing" on a health endpoint, which a
multi-tenant policy plane absolutely needs.

### 1.6 Which Package Should This Project Use?

**Use `opa/v1/sdk`, and reach for `opa/v1/rego` only for a second, separate concern.**

The reasoning is not about evaluation — both end up in the same `topdown` evaluator with the same
latency. It is that requirements 3 and 5 of this project are *exactly* the management-plane
features the SDK exists to provide. Writing bundle polling, ETag handling, signature
verification, persistence-to-disk-for-restart, status reporting and decision-log batching by hand
against `opa/v1/rego` means reimplementing several thousand lines of audited, CVE-tracked OPA
plugin code, including a JWT-based signature verification path where a mistake is a silent
integrity failure. For a project whose entire thesis is policy integrity, hand-rolling the
signature verification is the wrong trade.

The cost of the SDK is honest and worth stating: it pulls in `plugins`, `server`, discovery, and
the whole configuration surface, so the binary is larger and the attack surface is wider than
`rego` alone. You are also configuring it with a JSON blob rather than typed Go, which pushes
config errors from compile time to startup time. Mitigate that by building the config document
from a typed Go struct you own and marshalling it, so your own config stays type-checked.

Where `opa/v1/rego` still earns its place: **ad-hoc evaluation of a policy that is not part of the
distributed bundle** — for example a `--dry-run` mode that lets a platform engineer test a
candidate policy file against a recorded PR diff locally. That path wants no management plane at
all and `rego.New` + `PrepareForEval` is exactly right.

---

## 2. Performance in Practice

### 2.1 The numbers that exist

OPA's documentation is thinner on absolute figures than one would like, and it contains **no
performance comparison at all between the default (topdown), IR, and WASM targets**. What it does
publish is `opa bench` output for its own example RBAC policy, which is the most useful concrete
anchor available ([Policy Performance](https://www.openpolicyagent.org/docs/policy-performance)):

```
$ opa bench --data rbac.rego 'data.rbac.allow'

| samples                                   |      27295 |
| ns/op                                     |      45032 |
| B/op                                      |      20977 |
| allocs/op                                 |        382 |
| histogram_timer_rego_query_eval_ns_median |      35846 |
| histogram_timer_rego_query_eval_ns_90%    |      44780 |
| histogram_timer_rego_query_eval_ns_99%    |     133936 |
| histogram_timer_rego_query_eval_ns_99.9%  |     335906 |
| histogram_timer_rego_query_eval_ns_max    |     336493 |
```

Read that carefully, because it is the whole latency answer:

- **Median evaluation is ~36 µs. Mean ~40 µs. Total `ns/op` ~45 µs.** Against a 500 ms budget
  this is four orders of magnitude of headroom. "Sub-millisecond" is comfortably achievable.
- **But the p99.9 is ~336 µs — roughly 9× the median.** That is garbage-collection and allocation
  noise, not policy complexity, and it is the tail you will actually see in production. Still well
  inside budget, but it means "sub-millisecond" should be stated as a p99 target, not a mean.
- **~21 KB and ~382 allocations per evaluation.** Evaluation is allocation-heavy. At webhook
  rates this is nothing; it is the reason the tail exists.

Elsewhere on the same page, benchmark output for individual RBAC rules shows ~26–27 µs/op at
~12 KB/op and ~230 allocs/op, so the order of magnitude holds across policy shapes.

A PR-diff policy is structurally different from RBAC — it iterates over a list of changed files
rather than looking up a user's roles — so **benchmark the real policy against a real large diff.**
`opa bench` is the tool and it reports `B/op` and `allocs/op` alongside time.

### 2.2 What dominates evaluation cost

From [Policy Performance](https://www.openpolicyagent.org/docs/policy-performance), the cost model
is about *policy structure*, not about the engine:

- **Iteration and search are the expensive operations.** Adding more *rules* barely moves
  evaluation time; adding more *iteration over large collections* does.
- **Object/set lookup is constant-time; array scanning is O(n).** A policy that does
  `some file in input.files; file.path == target` scans; one that does `input.files_by_path[target]`
  does not. For a diff with thousands of changed files this is the difference that matters.
- **Rule indexing** (ground prefix indexing) prunes candidate rules before evaluating them, using
  equality statements on non-nested references, glob patterns with complete delimiter-separated
  segments, prefix/suffix matches on compile-time-known strings, and `in` membership checks. The
  docs note a real limit: *"past 32 for a single rule only the conditions common to every
  combination are indexed."*
- **Early exit**: for rules producing a single ground value (`allow if {...}`), OPA stops once one
  match is found. Controlled at the Go level by `rego.EvalEarlyExit(bool)`.
- **Comprehension indexing** memoizes comprehension results to turn an O(n²) group-by into O(n),
  but only under strict conditions — the comprehension must close over outer variables and must
  not appear inside `walk()`, inside a nested comprehension, or under a `with` statement.

The profiling tools: `opa eval --profile` reports per-expression `total_time_ns`, `num_eval`,
`num_redo`, and `num_gen_expr`, sortable by any of those. `opa bench` gives the aggregate
histogram shown above. Both are CLI; from Go, `rego.Metrics`/`rego.Instrument` and
`rego.EvalMetrics` expose per-phase timers (`timer_rego_query_eval_ns` and friends) that you can
export to Prometheus.

### 2.3 Concurrency

Three facts, which together fully determine the design:

1. **A `PreparedEvalQuery` is safe to share across goroutines** — documented explicitly, and
   structurally evident from the value receiver on `Eval`.
2. **A single evaluation is single-threaded.** From the docs: *"Policy evaluation is currently
   single-threaded. If you are embedding OPA as a library, it is your responsibility to dispatch
   concurrent queries to different Goroutines/threads."* One decision will not use more than one
   core. This is fine — the goal is 36 µs on one core, not parallel speedup.
3. **Throughput scales by running concurrent `Eval` calls**, bounded by `GOMAXPROCS`.

So: one prepared query, many goroutines, and **bound the concurrency yourself**. A webhook flood
that spawns an unbounded goroutine per PR will exhaust memory long before it exhausts CPU, given
21 KB of allocation per evaluation and no per-evaluation memory cap (see §4.3). A semaphore sized
to a small multiple of `GOMAXPROCS`, with requests beyond it rejected rather than queued
indefinitely, is the correct shape.

### 2.4 Memory behaviour

From [Policy Performance → Resource Utilization](https://www.openpolicyagent.org/docs/policy-performance):

- **Raw JSON loaded into OPA uses roughly 20× the memory of the same data serialized on disk**,
  because it is materialized into Go maps, slices and strings. The worked example: 8 MB of JSON
  representing 100,000 permission objects consumes ~160 MB of RAM.
- **Memory scales linearly with rule count**: ~10,000 ACL-style rules ≈ 130 MB; 100,000 rules
  ≈ 1.1 GB.
- Policy and data are **in-memory by default**; a disk-backed store exists
  ([Storage](https://www.openpolicyagent.org/docs/storage)) if the `data` document ever outgrows
  RAM.

For a multi-tenant deployment this is the figure to design around. If every tenant's policy is a
separate compiled bundle held simultaneously, memory is `tenants × (policy + data)`, and the 20×
JSON inflation applies to any per-tenant data you load (org charts, CODEOWNERS expansions, service
catalogues). Two mitigations worth considering early: keep per-tenant `data` small and push bulk
lookups to a built-in or a resolver instead of loading them into `data`; and consider a single
shared policy bundle parameterized by tenant — which is precisely the Gatekeeper
ConstraintTemplate/Constraint split discussed in §6.

Two environment variables, both documented by OPA as operational requirements rather than
suggestions:

- **`GOMEMLIMIT`** — set it slightly below the container's hard memory limit. The docs explain
  why precisely: without it, *"the heap can grow to roughly twice the live heap before GC runs
  (the `GOGC=100` default)"*, and when OPA holds a lot of data that doubling causes an OOM kill
  before GC catches up. Setting it too close to the live heap has the opposite cost — more
  frequent GC, more CPU, higher latency.
- **`GOMAXPROCS`** — limits cores. Since Go 1.25 OPA auto-detects the cgroup CPU quota and sets
  it, with a minimum of 2. The docs warn that with `GOMAXPROCS=1` you should benchmark GC
  behaviour against your latency requirements.

### 2.5 The WASM and IR targets: not worth it here

`opa build -t` accepts `{rego, wasm, plan}` ([CLI Reference](https://www.openpolicyagent.org/docs/cli)).
`rego` is the default (source in the bundle, compiled by the consumer); `wasm` emits a
`policy.wasm` module; `plan` emits an IR evaluation plan.

**WASM.** The target compiles a policy to a WASM module with a declared entrypoint:
`opa build -t wasm -e example/allow example.rego`. The module wraps results as
`[{ "result": <value> }]`, at most one assignment per evaluation. It requires a host ABI: shared
memory as `env.memory`, and imports `opa_abort`, `opa_println`, and `opa_builtin0` through
`opa_builtin4` for dispatching built-ins the module cannot implement itself; exported constants
`opa_wasm_abi_version` and `opa_wasm_abi_minor_version` carry the ABI version
([WASM docs](https://www.openpolicyagent.org/docs/wasm)).

Four reasons to skip it for this project:

1. **It is behind a build tag.** `rego.Target("wasm")` only works if the binary is built with the
   `opa_wasm` build tag — verified from the build constraint at the top of
   [`v1/rego/rego_wasmtarget_test.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/rego/rego_wasmtarget_test.go):
   `//go:build opa_wasm`. A non-default build tag in your own build pipeline is friction, and it
   means the code path is less exercised than the default one.
2. **Built-in coverage is incomplete.** The docs say there are *"a number of built-in functions
   that are not, and probably won't be natively supported in Wasm (e.g., `http.send`)"* and
   notably **do not publish a complete list** of what is missing. Discovering a gap at runtime, in
   a tenant's policy, is a bad failure mode.
3. **The performance claim is unquantified.** The integration page's comparison table rates WASM
   evaluation "Fastest" versus Go Lib "Faster", and says WASM *"should outperform the Go API
   (because the policies have been compiled to a lower-level instruction set)"* — note "should".
   There is no published benchmark. Trading documented, exercised behaviour for an unmeasured
   speedup, when you already have ~36 µs against a 500 ms budget, is a bad trade.
4. **The benefit is for other languages, not Go.** WASM's real purpose is embedding Rego in
   runtimes that have no Go. This project is Go.

One point of interest for the sandboxing question in §4: **OPA's WASM runtime is now
[wazero](https://github.com/tetratelabs/wazero)**, a pure-Go WASM runtime with no CGO —
`github.com/tetratelabs/wazero v1.12.0` appears in OPA's `go.mod`, and
`internal/wasm/sdk/internal/wasm/vm.go` imports `wazero`, `wazero/api` and `wazero/sys`. Older OPA
versions used wasmtime-go, which required CGO; that is no longer true, so the WASM target no
longer compromises static-binary builds. This does not change the recommendation, but it does
remove the historical objection, and it is relevant to §7's discussion of WASM as a generic
sandbox.

**IR.** The intermediate representation is *"suitable for custom compilers and evaluators"*
([IR docs](https://www.openpolicyagent.org/docs/ir)). Its purpose is to let people write
evaluators in languages OPA has no implementation for — the two cited consumers are
[swift-opa](https://github.com/open-policy-agent/swift-opa) and
[java-opa-sdk](https://github.com/open-policy-agent/java-opa-sdk). **There is no reason for a Go
service to touch it.** Go already has the reference evaluator.

**What to use instead for speed, if ever needed:** `opa build -O=1`. Optimization applies partial
evaluation to precompute known values, with the stated goal of *"convert[ing] non-linear-time
policies into linear-time policies"*. `-O=1` is documented as the recommended level: rules that
do not depend on unknowns are evaluated at build time and inlined; rules that do are partially
evaluated but not inlined, so the output stays structurally similar to the input. `-O=2` is
aggressive — it inlines virtual documents from rules that depend on unknowns, applies copy
propagation, and inlines certain negated statements. `opa build` automatically marks `input` as
unknown, and with `-b`/`--bundle` also marks any `data` references not prefixed by the
`.manifest` roots as unknown. `-O=0` (no optimization) is the default.

This is the right lever because it happens in the **bundle build pipeline**, not in the service —
which fits a project where policy is built centrally and shipped as a signed artifact. Note the
tradeoff: an optimized bundle is harder to relate back to the policy source a human wrote, which
matters for the audit trail in §5. Keep the unoptimized source addressable by bundle revision.

---

## 3. Policy Distribution: Bundles

This section is load-bearing for the project's thesis, so it is treated in full. Unless noted,
everything here is from [Bundles](https://www.openpolicyagent.org/docs/management-bundles).

### 3.1 The bundle format

A bundle is a **gzipped tarball** (`.tar.gz`) containing policies and/or data. Directory layout
determines where content lands in OPA's `data` document.

- **`.rego` files** load as modules, placed by their `package` declaration — `package example.authz`
  becomes `data.example.authz`. The file's path in the tarball is irrelevant to placement.
- **Data files**: *"OPA will only load data files named `data.json` or `data.yaml`."* Other
  JSON/YAML files are **silently ignored** — a sharp edge. Directory nesting maps to data paths:
  `roles/bindings/data.json` loads into `data.roles.bindings`.
- **WASM**: *"OPA will only load Wasm modules named `policy.wasm`."*
- **`.manifest`** — optional JSON at the bundle root.
- **`.signatures.json`** — present in signed bundles (§3.4).

The manifest fields:

| Field | Type | Meaning |
|---|---|---|
| `revision` | string | Bundle version identifier. A git commit hash is the idiomatic value. |
| `roots` | array | The scope of `data` and policy this bundle owns. |
| `rego_version` | integer | `0` or `1` — the Rego language version for the bundle's modules. |
| `file_rego_versions` | object | Per-file overrides, glob-keyed. |
| `metadata` | object | Arbitrary metadata, queryable from policy via `data.system`. |
| `wasm` | array | WASM module descriptors with `entrypoint` and `module`. |

```json
{
  "revision": "7864d60dd78d748dbce54b569e939f5b0dc07486",
  "roots": ["roles", "http/example/authz"],
  "wasm": [{"entrypoint": "http/example/authz/allow", "module": "path/to/policy.wasm"}]
}
```

**`roots` is the multi-tenancy primitive and the most under-appreciated feature here.** When a
manifest declares roots, OPA validates that the roots do not overlap, that every policy's package
is contained under a root, and that all data is contained under a root. More importantly, OPA can
run **multiple bundles simultaneously**, each owning a disjoint set of roots, and each updating
independently. A bundle cannot write outside its declared roots.

That is directly applicable: give each tenant (or each policy domain) its own bundle with its own
root, e.g. `roots: ["ci/tenant/acme"]`. Tenant A's bundle then *cannot* define rules or data under
tenant B's root — enforced by OPA, not by your code. And because `revision` is per-bundle, you get
independent versioning and independent rollback per tenant for free. This is a real, enforced
isolation boundary and it should shape the design.

`metadata` is the place to put provenance that policy can read — source repository, commit,
builder identity, build timestamp — accessible from Rego at `data.system.bundles[name].manifest.metadata`.
Also useful to surface on a status endpoint.

### 3.2 The Bundle Service API and polling

OPA pulls; the server never pushes. The contract is deliberately minimal — **any static file
server, S3 bucket, or GitHub Pages site can serve bundles.**

OPA constructs the URL as `<service.url>/<resource>`. With
`services[0].url = https://example.com/service/v1` and `resource = somedir/bundle.tar.gz`, it
fetches `https://example.com/service/v1/somedir/bundle.tar.gz`. If `resource` is omitted it
defaults to `bundles/<bundle-name>`.

Caching uses standard HTTP: OPA sends `If-None-Match: <last-etag>` (RFC 7232). The server returns
**200** with the tarball if changed, **304 Not Modified** if not. *"Services implementing the
Bundle Service API should set the HTTP `Etag` header in bundle responses to identify the
revision."* So polling a 50 KB bundle every 20 seconds costs one 304 round trip, not a download.

Configuration — these keys go into the `Config io.Reader` you hand `sdk.New`:

```yaml
services:
  - name: acmecorp
    url: https://example.com/service/v1
    credentials:
      bearer:
        token: "..."

bundles:
  authz:
    service: acmecorp
    resource: somedir/bundle.tar.gz
    persist: true
    polling:
      min_delay_seconds: 10
      max_delay_seconds: 20
    signing:
      keyid: my_global_key
      scope: read
```

**Short polling** (the default) issues requests at a random interval between `min_delay_seconds`
and `max_delay_seconds` — the jitter spreads load across many OPA instances.

**Long polling** is enabled by `polling.long_polling_timeout_seconds`. The server signals support
by responding with `Content-Type: application/vnd.openpolicyagent.bundles`; if it does not, OPA
*"will fallback to the regular periodic polling"*. This is how you get near-immediate policy
propagation without hammering the server.

**`persist: true`** writes the activated bundle to disk at
`./.opa/bundles/<bundle-name>/bundle.tar.gz` and *"OPA will attempt to read the bundle from disk
on startup"*. This matters more than it sounds for this project: without it, a service restart
while the policy server is unreachable comes up **with no policy at all**, and a PR-gating service
with no policy must either block every PR or allow every PR. With `persist: true` it comes up on
the last known good policy. **Set it.**

**Manual triggering**: `bundles[_].trigger: manual` disables periodic polling, driven instead from
Go via the bundle plugin's `Trigger(ctx)` (see §1.5). For a policy that changes on merge to a
policy repository, an event-driven trigger from that repository's webhook is a better fit than
polling — and OPA's docs caution to *"do this sparingly, it is not intended to be used as a
replacement for periodic bundle polling."* The robust pattern is both: slow polling as the
backstop, manual trigger for immediacy.

### 3.3 Delta bundles

A delta bundle carries **JSON Patch operations against `data`** instead of a full snapshot, for
efficiency when data changes often.

Structure: `patch.json` at the bundle root (required), plus an optional `.manifest`.
**No policy, data, or wasm files are allowed.**

```json
{
  "data": [
    { "op": "upsert", "path": "/a/b", "value": ["hello", "world"] },
    { "op": "remove", "path": "/a/c" },
    { "op": "replace", "path": "/x", "value": 42 }
  ]
}
```

| `op` | Behaviour |
|---|---|
| `upsert` | Create or update at path; inserts if the path is an array index. Requires `value`. |
| `replace` | Replace an existing value. Requires `value`; path must exist. |
| `remove` | Delete at path; path must exist. `value` ignored. |

`path` is an RFC 6902 JSON Pointer. If a previously activated snapshot declared `roots` or `wasm`,
the delta's manifest *"must have the same values"* for those fields; `revision` and `metadata` may
differ.

**Three limitations, and they rule delta bundles out for this project's policy path:**

- **Data only — policies cannot be patched.**
- ***"Do not support bundle signing."***
- ***"Activated delta bundles are not persisted to disk."***

The signing limitation is decisive. Policy must be signed, so policy ships as snapshot bundles.
Delta bundles remain available for high-churn *reference data* where integrity is less critical —
but given that such data feeds security decisions, the honest answer is to use snapshot bundles
throughout and accept the bandwidth, which for a policy artifact measured in kilobytes is
irrelevant.

### 3.4 Bundle signing and verification

This is the mechanism that makes "policy lives outside the repository it governs" a *technical*
guarantee rather than a procedural one, so it is worth understanding end to end.

**The artifact.** A signed bundle contains `.signatures.json` at its root:

```json
{
  "signatures": [
    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJmaWxlcyI6W..."
  ]
}
```

Each entry is a **JWT**. Its payload lists every other file in the bundle with a cryptographic
hash:

```json
{
  "files": [
    {
      "name": ".manifest",
      "hash": "c2131544c716a25a5e31f504300f5240e8235cadb9a57f0bd1b6f4bd74b26612",
      "algorithm": "SHA-256"
    }
  ],
  "scope": "write",
  "iat": 1592248027,
  "iss": "JWTService",
  "keyid": "myPublicKey"
}
```

The JWT header carries `alg`, `typ: "JWT"`, and optionally `kid` — *"Key ID (determines
verification key if present)"*.

A detail that matters for reproducible builds: for JSON and YAML files the hash is not over raw
bytes. OPA *"read[s] the byte stream and parse[s] into a JSON structure; then recursively order[s]
the fields of all objects alphabetically"* before hashing. Unstructured files hash their raw bytes.
So JSON key order and whitespace do not break signatures. Supported hash algorithms: MD5, SHA-1,
SHA-224, SHA-256, SHA-384, SHA-512, SHA-512-224, SHA-512-256. **Use SHA-256 or better; MD5 and
SHA-1 being accepted is a legacy allowance, not a recommendation.**

**Verification, and this is the part to get right.** On receiving a signed bundle OPA:

1. *"Verif[ies] the JWT signature with the appropriate public key"*
2. *"Verif[ies] that the JWT payload and target directory specify the same set of files"*
3. *"Verif[ies] the content of each file by checking the hash recorded in the JWT payload"*

Step 2 is the one people forget to appreciate: it means an attacker cannot **add** an unsigned
file to a signed bundle, nor **remove** a signed one. Signature covers the file *set*, not just
each file's content. For a policy bundle, deleting the rule that blocks you is as good as editing
it, so this closes the obvious attack.

And the failure behaviour is already fail-closed: OPA *"activates the new bundle only if all the
verification steps succeed; otherwise, **it continues using its existing bundle** and reports an
activation failure."* It does not fall back to no policy. Combined with `persist: true`, this is
exactly the behaviour this project needs — but it also means **a signature failure is silent from
the decision path's point of view.** The service keeps happily enforcing stale policy. You must
watch the Status API / bundle status listener for activation failures and alert on them, or a
compromised or misconfigured policy pipeline will go unnoticed indefinitely. This is the single
most important operational alert in the system.

**Configuring keys** — public keys go in a top-level `keys` section:

```yaml
keys:
  my_global_key:
    algorithm: RS256
    key: |
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
      -----END PUBLIC KEY-----
```

and are referenced per bundle:

```yaml
bundles:
  authz:
    service: acmecorp
    resource: bundle.tar.gz
    signing:
      keyid: my_global_key
      scope: read
```

Key lookup order is: command-line `--verification-key` (for filesystem bundles) → the config
`keys` section (for remote bundles) → the JWT `kid` header.

**Use RS256 (asymmetric), not HS256.** OPA accepts an HMAC secret for `--signing-key`, but a
symmetric key means every verifying service holds the key that can *forge* bundles — which
destroys the project's threat model, since the verifier is the thing running in the environment
the policy constrains. With RS256 the private key lives only in the central build pipeline and
each service holds only the public key.

**`scope` is an underused authorization control.** The `scope` claim in the JWT payload must match
the `signing.scope` configured out-of-band. This lets one signing identity be constrained to one
purpose: a key authorized for `scope: "ci-security-policy"` cannot sign a bundle that a service
expecting `scope: "ci-test-selection"` will accept. Given this project's split between
fail-closed security policy and fail-open test-selection policy, **signing those two bundle
families with different scopes (ideally different keys) is a cheap and strong boundary.**

**Building and signing** ([CLI Reference](https://www.openpolicyagent.org/docs/cli)):

```bash
opa build --signing-key /path/to/private_key.pem \
          --verification-key /path/to/public_key.pem \
          --bundle foo/
```

Verified flags on `opa build`: `--bundle`/`-b`, `--target`/`-t` (`{rego, wasm, plan}`),
`--entrypoint`/`-e`, `--optimize`/`-O`, `--output`/`-o`, `--capabilities`, `--signing-key`,
`--verification-key`, `--signing-alg`, `--claims-file`, `--signing-plugin`, `--revision`/`-r`.

`--claims-file` supplies the optional JWT claims (`scope`, `iss`, `keyid`):

```bash
opa build --claims-file claims.json --signing-key ... --bundle foo/
```

`--revision`/`-r` sets the manifest revision — wire it to the policy repo's git SHA in CI.

There is also a standalone `opa sign` command — *"Generate an opa bundle signature"* — producing
`.signatures.json` without rebuilding, with flags `--bundle`/`-b`, `--signing-key`,
`--signing-alg`, `--claims-file`, `--signing-plugin`, `--output-file-path`/`-o`. This is the hook
for signing in a **separate, more privileged CI step** than the one that builds, so the build job
never holds the private key.

`--signing-plugin` deserves a flag of its own: it is the extension point for custom
signing/verification. **Unverified**: I did not confirm whether an off-the-shelf plugin exists for
KMS-backed or Sigstore/cosign-based signing, nor the plugin interface's Go signature. If the
project wants keyless signing with an OIDC identity (which would be the modern answer for an
open-source project — no long-lived private key to steal), this is the thing to investigate, and
it should be investigated before committing to the long-lived-RSA-key design.

For filesystem bundles the verification key is given at run time:

```bash
opa run --verification-key /path/to/public_key.pem bundle.tar.gz
```

### 3.5 OCI registries as the distribution channel

OPA can pull bundles directly from an OCI registry by setting `type: oci` on the service:

```yaml
services:
  ghcr-registry:
    url: https://ghcr.io
    type: oci
    credentials:
      bearer:
        scheme: Bearer
        token: "<mytoken>"

bundles:
  authz:
    service: ghcr-registry
    resource: ghcr.io/someorg/policy-hello:1.0.0
    persist: true
```

A policy image has three layers: a manifest layer, the bundle tarball layer
(`application/vnd.oci.image.layer.v1.tar+gzip`), and a currently-empty config layer. Publishing
uses `opa build` plus ORAS:

```bash
opa build <path_to_src>

oras push <registry>/<org>/<repo>:<tag> \
  --manifest-config config.json:application/vnd.oci.image.config.v1+json \
  bundle.tar.gz:application/vnd.oci.image.layer.v1.tar+gzip
```

An empty `{}` config.json suffices. Accepted media types are
`application/vnd.oci.image.layer.v1.tar+gzip`, `.../manifest.v1+json`, `.../config.v1+json`,
`.../index.v1+json`, and `application/vnd.docker.distribution.manifest.list.v2+json`. For image
indexes, *"OPA descends into the index and uses the first referenced manifest that has an
`application/vnd.oci.image.layer.v1.tar+gzip` layer"*, skipping provenance and SBOM entries.
Limitation: *"Currently only one layer per image"* containing the bundle. Downloaded images are
cached in an `oci` folder under the persistence directory, which *"maintain/clean periodically"*.

**This is probably the right distribution channel for this project**, and the reasoning is about
ecosystem rather than about OPA. An OCI registry gives you, for free and already audited:
immutable content-addressed digests, tags for environment promotion (`:stable`, `:canary`),
authentication and per-repository authorization you do not write, replication and pull-through
caching, and — the big one — the entire **supply-chain tooling ecosystem**: cosign signatures and
attestations, SBOMs, provenance, and admission-time verification policies. A self-hosting user
already runs a registry or has access to GHCR.

Two caveats to state plainly. First, OPA's bundle signing and OCI/cosign signing are **independent
mechanisms** operating at different layers — OPA verifies `.signatures.json` inside the tarball;
cosign verifies the image manifest digest. They are complementary, not redundant, and OPA itself
will not check a cosign signature. Second, note that OPA's `credentials.bearer.token` is a static
token in a config file; registry tokens usually expire. OPA supports other credential types for
services (**unverified**: I did not confirm which OCI-compatible credential helpers or
token-refresh schemes OPA's service configuration supports — check the
[Configuration](https://www.openpolicyagent.org/docs/configuration) reference before designing
registry auth).

### 3.6 How policy should actually reach this service

Synthesising the above into a concrete recommendation:

1. Policy lives in a **separate git repository** with its own branch protection and CODEOWNERS —
   a repository the governed developers have no write access to. This, not any OPA feature, is
   where the project's thesis is actually enforced. OPA's role is to make tampering *downstream*
   of that repository detectable.
2. CI in that repository runs `opa check` and the policy's own tests, then
   `opa build -b . -O=1 -r "$GIT_SHA"`, then — in a **separate job holding the signing key, or
   better, a keyless OIDC identity** — `opa sign` with a `scope` claim naming the policy family.
3. The signed bundle is pushed to an **OCI registry** and tagged for the target environment.
4. Each control-plane instance runs the **OPA SDK** configured with `type: oci`, the matching
   `keys` entry, `signing.keyid` and `signing.scope`, `persist: true`, long polling (or manual
   trigger from a repository webhook plus slow polling as a backstop).
5. The service exposes the **active bundle revision** on a status endpoint, sourced from the bundle
   plugin's status listener, and **alerts on activation failure** — because a failed signature
   verification leaves the service silently enforcing stale policy.
6. Per-tenant isolation uses **bundle `roots`**, so OPA structurally prevents one tenant's policy
   from defining rules under another's namespace.

---

## 4. Guardrails Against Hostile or Broken Policy

This is where the honest answer is least comfortable, so it comes first: **OPA has a good story for
bounding *time* and a good story for restricting *capability*, and essentially no story for
bounding *memory or allocation*.** For a project that runs tenant-authored code, that gap has to be
closed outside OPA.

### 4.1 Timeouts and cancellation: solid

Cancellation is driven entirely by `context.Context`. `rego.PreparedEvalQuery.Eval(ctx, ...)`
translates `ctx` into OPA's internal `topdown.Cancel` (the goroutine shown in §1.2), and the
evaluator checks it at the top of **every expression evaluation**. From
[`v1/topdown/eval.go#L371`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/topdown/eval.go):

```go
func (e *eval) evalExpr(iter evalIterator) error {
	// ...
	if e.cancel != nil && e.cancel.Cancelled() {
		cancelErr := &Error{Code: CancelErr, Message: "caller cancelled query execution"}
		if e.ctx != nil && e.ctx.Err() != nil {
			err := e.ctx.Err()
			cancelErr.Message = err.Error()
			cancelErr.err = err
		}
		return cancelErr
	}
	// ...
}
```

This is the good news, and it is better than it looks. Because the check is per-expression rather
than per-rule or at some polling interval, **any** non-terminating Rego — infinite comprehension,
pathological recursion through `walk`, a quadratic join over a huge array — passes through
`evalExpr` and will be interrupted. A `context.WithTimeout` is therefore a *reliable* bound on
evaluation wall time.

```go
ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
defer cancel()
resultSet, err := prepared.Eval(ctx, rego.EvalInput(input))
```

Two residual gaps:

- **A blocking built-in is only interruptible if it honours `ctx` itself.** `http.send` in a
  tenant's policy could stall. The correct fix is not a timeout — it is removing the built-in
  entirely (§4.2).
- **A single expression that allocates enormously will not be interrupted mid-allocation.** The
  check happens *between* expressions. One comprehension that materializes a huge intermediate set
  can exhaust memory before control returns to `evalExpr`. This is the real hole, and §4.3 is
  about it.

### 4.2 Restricting capability: `rego.Capabilities` is the main tool

`ast.Capabilities` is the allowlist, verified from
[`v1/ast/capabilities.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/ast/capabilities.go):

```go
type Capabilities struct {
	Builtins        []*Builtin       `json:"builtins,omitempty"`
	FutureKeywords  []string         `json:"future_keywords,omitempty"`
	WasmABIVersions []WasmABIVersion `json:"wasm_abi_versions,omitempty"`
	Features        []string         `json:"features,omitempty"`
	AllowNet        []string         `json:"allow_net,omitempty"`
}
```

with constructors `ast.CapabilitiesForThisVersion(opts ...CapabilitiesOption) *Capabilities`,
`ast.LoadCapabilitiesJSON(io.Reader)`, `ast.LoadCapabilitiesFile(string)`, and
`ast.LoadCapabilitiesVersion(string)`. `ast.Builtin` has a `Name string` field and, usefully, a
`Nondeterministic bool` field.

The mechanism, and why it is the right one: **`Builtins` is an allowlist checked at compile time.**
A policy referencing a built-in not in the list fails `PrepareForEval` with a
`rego_type_error`. That is enormously better than a runtime check — a tenant's hostile policy is
**rejected at upload/activation time, not at decision time**, so it never reaches the decision
path at all. Validate tenant policy at bundle-build time with `opa build --capabilities <file>`
*and* at load time in the service with `rego.Capabilities(...)`, so the second is a real barricade
rather than a duplicate of the first.

Build a restricted set by filtering the defaults:

```go
// restrictedCapabilities returns OPA's capabilities for this version with every
// nondeterministic built-in and every explicitly denied built-in removed, and
// with all network access forbidden.
func restrictedCapabilities() *ast.Capabilities {
	denied := map[string]struct{}{
		"http.send":          {},
		"net.lookup_ip_addr": {},
		"opa.runtime":        {},
		"rego.parse_module":  {},
		"trace":              {},
	}

	capabilities := ast.CapabilitiesForThisVersion()
	allowed := make([]*ast.Builtin, 0, len(capabilities.Builtins))
	for _, builtin := range capabilities.Builtins {
		if _, isDenied := denied[builtin.Name]; isDenied {
			continue
		}
		if builtin.Nondeterministic {
			continue
		}
		allowed = append(allowed, builtin)
	}
	capabilities.Builtins = allowed

	// A non-nil, empty AllowNet forbids all network access. A nil AllowNet
	// permits ANY host -- the difference is load-bearing.
	capabilities.AllowNet = []string{}

	return capabilities
}
```

Dropping all nondeterministic built-ins is worth doing on its own merits, beyond security: a
PR-gating decision should be **reproducible**. If the same diff and the same policy revision can
produce different verdicts because the policy called `time.now_ns()` or `rand.intn()`, the audit
trail in §5 is worthless and "why was my PR blocked?" becomes unanswerable. Determinism is an
audit requirement here, not just a hardening measure.

**`AllowNet` — and a documentation discrepancy worth recording.** The struct's own doc comment in
v1.21.2 reads:

```go
// allow_net is an array of hostnames or IP addresses, that an OPA instance is
// allowed to connect to.
// If omitted, ANY host can be connected to. If empty, NO host can be connected to.
// As of now, this only controls fetching remote refs for using JSON Schemas in
// the type checker.
```

That last sentence is **stale**. `AllowNet` *does* gate `http.send`: `v1/topdown/http.go` defines
`verifyHost(caps *ast.Capabilities, host string)` and `verifyURLHost(caps, unverifiedURL)`, and
calls `verifyURLHost(bctx.Capabilities, ...)` on the request URL (lines ~402–411, ~475, ~648 of
[`v1/topdown/http.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/topdown/http.go)).
The documentation site is correct and the code comment is out of date. Recording this because the
comment is exactly the kind of thing that would otherwise lead someone to wrongly conclude
`AllowNet` is not a real egress control.

Note the semantics precisely, because they are inverted from intuition: **`nil` permits every
host; empty-but-non-nil permits none.** `AllowNet: []string{}` is the lockdown; omitting the field
is wide open.

**A known bypass.** [open-policy-agent/opa#9320](https://github.com/open-policy-agent/opa/issues/9320)
(reported 2026-10-02, **open** as of 2026-10-13, PR #9324 referenced, status unclear) reports that
`allow_net` does not restrict `unix://` destinations in `http.send`: with
`allow_net: ["allowed.example.com"]`, a policy can reach any local UNIX socket via
`unix://allowed.example.com/path?socket=/arbitrary/socket/path`, because the socket path arrives in
a query parameter and is never re-checked. In this project's threat model — tenant-authored policy
running in a service that may sit near a Docker socket or a cloud metadata endpoint — that is
serious. **It is also entirely mooted by removing `http.send` from the allowlist**, which is what
the code above does and what this project should do regardless. Defence in depth: do not grant the
built-in, *and* set `AllowNet: []string{}`.

**This recommendation is independently corroborated by the most mature system in the space.**
Gatekeeper *"disables the `http.send` built-in function by default"* as of v3.8.0, via
`--disable-opa-builtin`, and replaced it with a mediated `external_data` protocol — explicitly
because *"if template authors are not trusted, it will potentially give template authors access to
the in-cluster network."* §6.1 covers the mechanism and why it should be copied wholesale, along
with a second compile-time control this section does not cover: restricting which `data` paths
tenant policy may read at all.

**`rego.UnsafeBuiltins(map[string]struct{})` is the weaker alternative.** Its own doc comment
explains why: *"This option is ignored for module compilation if the caller supplies the compiler.
This option is always honored for query compilation."* A denylist that is silently ignored under a
common configuration is not a security boundary. Prefer `Capabilities`. A denylist is also the
wrong shape — every new OPA release can add built-ins, and a denylist fails open on anything it
has not heard of, whereas an allowlist fails closed.

Finally, `opa capabilities` (`--current`, `--version`, `--file`) prints capability sets, which is
how you generate a baseline file to check into the policy repository.

### 4.3 Memory and non-termination: the real gap

**There is no per-evaluation memory limit, allocation limit, or instruction/step limit in OPA.**
This was verified by enumerating every option on the evaluator. `v1/topdown/query.go` exposes 37
`With*` methods — `WithCancel`, `WithEarlyExit`, `WithIndexing`, `WithStrictBuiltinErrors`,
`WithVirtualCache`, `WithBaseCache`, and so on — and **none** of them bounds memory, allocations,
or evaluation steps. A grep for `memory`, `maxstep`, `budget` and `limit` across that file returns
one unrelated comment. The `rego` package likewise exposes no such option.

Compare CEL, which has a first-class interpreter cost budget (§7.1). This is a genuine
architectural difference between the two engines, and for a project running untrusted policy it is
the strongest single argument *against* OPA.

What this means concretely: a tenant can write a terminating, finite, perfectly legal Rego policy
that allocates gigabytes — a comprehension crossing two moderately large collections is enough —
and **OOM-kill the whole multi-tenant control plane.** The `context` timeout does not save you,
because the allocation can happen inside one expression, and because dying of OOM is faster than
50 ms.

Mitigations, in rough order of value:

1. **Bound the input.** This is the highest-value control and the one fully under your command. A
   quadratic blow-up needs a large collection to be quadratic *in*. Cap the number of changed
   files, the size of each patch, path depth, and total input bytes **before** evaluating, and
   reject — never truncate, because a truncated diff is one the policy never actually saw. Most
   pathological policies become harmless against a 1,000-file cap.
2. **Set `GOMEMLIMIT`** below the container limit (§2.4). This makes Go GC work harder instead of
   the kernel OOM-killing you, converting some OOM kills into slow-but-alive degradation. It is a
   softener, not a bound.
3. **Bound concurrency** (§2.3). N concurrent evaluations multiply peak memory by N.
4. **Validate policy at admission, not at decision time.** Capabilities checking, `opa check
   --strict`, and running the tenant's policy against a corpus of recorded diffs in a *sacrificial*
   process before accepting the bundle. This catches the accidental pathological policy, which is
   the common case, though not a deliberately hostile one.
5. **Process isolation for genuinely hostile multi-tenancy.** If tenants are mutually distrusting
   and some may be actively malicious, the only sound answer is that one tenant's evaluation cannot
   share a heap with another's: evaluate in a subprocess with an RLIMIT/cgroup memory cap, or inside
   a WASM sandbox with a memory limit and fuel metering (wazero — already in OPA's dependency tree,
   §2.5 — supports both). This is a significant architectural cost, and it is the price of the
   requirement "end users author the policies" taken literally.

**Be clear-eyed about which threat model applies.** If tenants are teams inside one organisation
who can be held accountable and whose policy goes through a reviewed central repository, controls
1–4 are proportionate and the residual risk is acceptable. If tenants are mutually anonymous users
of a hosted service, control 5 is mandatory and OPA-in-process is the wrong architecture
regardless of engine. The project description says policy is authored by end users *and*
distributed from a central developer-inaccessible source — which implies the former, and implies
the central repository's review process is itself a load-bearing security control. **Say so
explicitly in the design, because it is easy to lose that assumption later.**

### 4.4 Distinguishable error modes

The requirement — fail closed on security decisions, fail open on test-selection decisions —
demands telling "the policy said no" from "the engine broke" from "I ran out of time". OPA
supports this well, provided you classify deliberately. Four distinct categories:

> **Read §6.1's fail-open/fail-closed discussion before settling this.** Gatekeeper ships a
> **three-way** split that is sharper than the two-way split below and should supersede it:
> *cannot reach the engine* → allow (availability); *engine ran and errored* → deny (correctness);
> *an attempt to self-exempt* → deny, always (integrity). It also argues that fail-closed is a
> **per-policy** decision rather than a system-wide one, which means the asymmetry belongs in
> policy metadata — not hard-coded into the two Go call sites shown at the end of this section.

**(a) Compile-time errors** (from `PrepareForEval`) surface as `ast.Errors`, which is
`[]*ast.Error` with a value-receiver `Error()` method, so `errors.As` works. Each `*ast.Error`
carries `Code`, `Message` and `Location` (with `Row`, `Col`, `File`). Verified codes from
[`v1/ast/errors.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/ast/errors.go):
`ParseErr = "rego_parse_error"`, `CompileErr = "rego_compile_error"`, `TypeErr = "rego_type_error"`,
`UnsafeVarErr = "rego_unsafe_var_error"`, `RecursionErr = "rego_recursion_error"`.

These are **policy-author bugs** and belong at bundle-admission time with a file/row/column
pointing at the author's mistake — never on the decision path. `RecursionErr` is notable: OPA
rejects recursive rules **at compile time**, which eliminates a whole class of non-termination
before evaluation ever starts.

**(b) Cancellation / timeout.** `topdown.IsCancel(err) bool` is the predicate. Verified from
[`v1/topdown/errors.go`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/topdown/errors.go):

```go
// IsCancel returns true if err was caused by cancellation.
func IsCancel(err error) bool {
	return errors.Is(err, cancelErr)
}
```

backed by an `Is` method on `*topdown.Error` that matches on `Code`. The corresponding code is
`topdown.CancelErr = "eval_cancel_error"`.

**(c) Runtime evaluation errors**, as `*topdown.Error` with a `Code` field. The full verified set:

| Constant | Value | Meaning |
|---|---|---|
| `topdown.InternalErr` | `eval_internal_error` | Unknown evaluation error — an OPA bug. |
| `topdown.CancelErr` | `eval_cancel_error` | Evaluation was cancelled. |
| `topdown.ConflictErr` | `eval_conflict_error` | Conflicting outputs, e.g. a complete rule producing two different values. |
| `topdown.TypeErr` | `eval_type_error` | Expression applied to a value of the wrong type. |
| `topdown.BuiltinErr` | `eval_builtin_error` | A built-in got invalid input or hit a runtime error. |
| `topdown.WithMergeErr` | `eval_with_merge_error` | Real and replacement data could not be merged. |

`*topdown.Error` also has a `StackTrace` field, populated only when enabled via
`rego.StackTraces(true)` / `topdown.Query.WithStackTraces`. The source comment notes it is
deliberately *"left out of the error message, so callers render it themselves"* — so enabling it
does not change your log lines, it just gives you a field. Worth enabling: when a tenant asks why
their policy errored, the stack of queries being evaluated is the answer.

`ConflictErr` is the one to watch in a multi-tenant system. It fires when two rules disagree, which
is exactly what happens when two independently-authored policy fragments both try to define the
same decision. Surface it as a *policy* error naming both locations, not as an internal error.

**(d) Undefined.** Not an error at all — an empty `ResultSet` from `rego`, or an error satisfying
`sdk.IsUndefinedErr` from the SDK. This is the easiest one to get wrong, because `len(rs) == 0`
looks like "nothing to see here" and silently becomes "allow".

A classifier that makes the fail-closed/fail-open decision explicit:

```go
// Sentinel errors distinguishing why a decision could not be produced.
var (
	ErrPolicyUndefined = errors.New("policy decision undefined")
	ErrPolicyTimeout   = errors.New("policy evaluation timed out")
	ErrPolicyBroken    = errors.New("policy evaluation failed")
	ErrPolicyInvalid   = errors.New("policy failed to compile")
)

// classifyEvalError maps an OPA evaluation error to one of this package's
// sentinel errors so callers can choose fail-closed or fail-open per decision.
func classifyEvalError(err error) error {
	if err == nil {
		return nil
	}
	if topdown.IsCancel(err) {
		return fmt.Errorf("%w: %v", ErrPolicyTimeout, err)
	}
	var astErrors ast.Errors
	if errors.As(err, &astErrors) {
		return fmt.Errorf("%w: %v", ErrPolicyInvalid, err)
	}
	var evalError *topdown.Error
	if errors.As(err, &evalError) {
		return fmt.Errorf("%w: %s: %v", ErrPolicyBroken, evalError.Code, err)
	}
	return fmt.Errorf("%w: %v", ErrPolicyBroken, err)
}
```

and at the two call sites, with the asymmetry stated in code rather than in a comment:

```go
// GateSecurity decides whether a PR may proceed. Any failure to produce a
// decision -- undefined, timeout, broken policy -- blocks the PR.
func (g *Gate) GateSecurity(ctx context.Context, diff any) (Decision, error) {
	decision, err := g.securityEngine.Evaluate(ctx, diff)
	if err != nil {
		return Decision{Allow: false, RequiresReview: true, Reasons: []string{err.Error()}}, err
	}
	return decision, nil
}

// SelectTests chooses which test suites to run. Any failure to produce a
// decision falls back to running everything, which is slow but never unsafe.
func (g *Gate) SelectTests(ctx context.Context, diff any) (TestPlan, error) {
	plan, err := g.selectionEngine.Evaluate(ctx, diff)
	if err != nil {
		return TestPlanRunAll(), nil
	}
	return plan, nil
}
```

Note that `SelectTests` **discards** the error from the caller's perspective but must still *log*
it — swallowing it entirely would mean a permanently broken test-selection policy degrades to
"run everything" forever with nobody noticing. Fail open on the decision, fail loud in the logs.

Two options that make errors *more* visible rather than less, and which this project should both
enable: `rego.StrictBuiltinErrors(true)` — *"tells the evaluator to treat all built-in function
errors as fatal errors"* — and `rego.Strict(true)` for compiler strict mode. By default, a
built-in error makes the expression **undefined** rather than failing, which silently turns a
broken policy into a permissive one. That default is wrong for this project in both directions:
on the security path it converts a malfunction into an allow, and on the audit path it erases the
evidence that anything went wrong. (The SDK exposes the same control as
`sdk.DecisionOptions.StrictBuiltinErrors`.) There is also `rego.BuiltinErrorList(*[]topdown.Error)`
if you want to collect built-in errors without making them fatal — useful for reporting
diagnostics on the fail-open path.

---

## 5. Decision Logging and Auditability

Decision logging is an SDK/plugin feature, not a `rego` feature — another reason for §1.6's
recommendation. The authoritative reference is
[Decision Logs](https://www.openpolicyagent.org/docs/management-decision-logs).

### 5.1 What gets recorded

The event struct is
[`v1/plugins/logs.EventV1`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/plugins/logs/plugin.go),
verified from source at v1.21.2:

```go
type EventV1 struct {
	Labels              map[string]string       `json:"labels"`
	DecisionID          string                  `json:"decision_id"`
	BatchDecisionID     string                  `json:"batch_decision_id,omitempty"`
	TraceID             string                  `json:"trace_id,omitempty"`
	SpanID              string                  `json:"span_id,omitempty"`
	Revision            string                  `json:"revision,omitempty"` // Deprecated: Use Bundles instead
	Bundles             map[string]BundleInfoV1 `json:"bundles,omitempty"`
	Path                string                  `json:"path,omitempty"`
	Query               string                  `json:"query,omitempty"`
	Input               *any                    `json:"input,omitempty"`
	Result              *any                    `json:"result,omitempty"`
	IntermediateResults map[string]any          `json:"intermediate_results,omitempty"`
	MappedResult        *any                    `json:"mapped_result,omitempty"`
	NDBuiltinCache      *any                    `json:"nd_builtin_cache,omitempty"`
	Erased              []string                `json:"erased,omitempty"`
	Masked              []string                `json:"masked,omitempty"`
	Error               error                   `json:"error,omitempty"`
	RequestedBy         string                  `json:"requested_by,omitempty"`
	Timestamp           time.Time               `json:"timestamp"`
	Metrics             map[string]any          `json:"metrics,omitempty"`
	RequestID           uint64                  `json:"req_id,omitempty"`
	RuleLabels          []map[string]any        `json:"rule_labels,omitempty"`
	RequestContext      *RequestContext         `json:"request_context,omitempty"`
	Custom              map[string]any          `json:"custom,omitempty"`

	inputAST ast.Value
}
```

For an audit trail answering "why was this PR blocked?", five fields carry the weight:

- **`Input`** — the exact PR diff structure evaluated. Without it the decision is unreproducible.
- **`Result`** — the decision returned, including reasons if the policy returns a rich document
  (which is the argument from §1.3 for binding a decision object rather than a boolean).
- **`Bundles[].Revision`** — *"Revision of the bundle at the time of evaluation"*. **This is the
  field that makes the audit trail complete**: input + policy revision + result is a reproducible
  triple. Given a git SHA in `revision` (§3.4, `opa build -r`), you can check out the exact policy
  and re-run the exact input. Nothing else in the system gives you that.
- **`DecisionID`** — *"Unique identifier generated for each decision for traceability"*. Post it
  into the GitHub check-run output so a developer reading "blocked by policy" has a handle to
  quote.
- **`Timestamp`**, and `TraceID`/`SpanID` which are W3C trace-context compliant, so decisions join
  your existing distributed traces.

Also worth noting: **`Error error`** means the log records *failed* decisions, not just successful
ones. Combined with §4.4's classification, this is how a permanently broken fail-open
test-selection policy becomes visible rather than invisible. And **`Custom map[string]any`** is the
place to attach your own correlation data — repository, PR number, tenant, installation ID — which
is essential in a multi-tenant deployment where `Labels` alone cannot distinguish tenants.
`RuleLabels` and the `ids` field (*"List of annotation `id` values for rules that were
successfully evaluated"*) let the log record **which rules fired**, which is a notably better
answer to "why was I blocked" than just the final boolean.

### 5.2 Configuration

Verified key-by-key from the
[Configuration reference](https://raw.githubusercontent.com/open-policy-agent/opa/v1.21.2/docs/docs/configuration.md)
at v1.21.2, including defaults. Note the nesting: the buffering and upload knobs live under
`decision_logs.reporting.*`, while the sink and policy-path knobs sit directly under
`decision_logs.*`.

| Key | Type | Default | Purpose |
|---|---|---|---|
| `decision_logs.service` | string | first service defined | Service to contact. Defaults to the first `services` entry if no `plugin` is set and `console` is disabled. |
| `decision_logs.resource` | string | `/logs` | Full path for sending logs to the remote server. |
| `decision_logs.console` | boolean | `false` | Log decisions locally to the console. |
| `decision_logs.plugin` | string | — | Use a named custom plugin. *"If this field exists, the other configuration fields are not required."* |
| `decision_logs.mask_decision` | string | `/system/log/mask` | Path of the masking decision. |
| `decision_logs.drop_decision` | string | `/system/log/drop` | Path of the drop decision. |
| `decision_logs.request_context.http.headers` | array | — | HTTP headers to include in the log, if present on the incoming request. |
| `decision_logs.partition_name` | string | — | **Deprecated — use `resource` instead.** |
| `decision_logs.reporting.buffer_type` | string | `size` | `size` or `event`. |
| `decision_logs.reporting.buffer_size_limit_events` | int64 | `10000` | Buffer limit in events. `event` buffer type only. |
| `decision_logs.reporting.buffer_size_limit_bytes` | int64 | unlimited | Buffer limit in bytes. `size` buffer type only. Mutually exclusive with `max_decisions_per_second`. |
| `decision_logs.reporting.max_decisions_per_second` | float64 | — | Rate limit; drops events above it. Mutually exclusive with `buffer_size_limit_bytes`. |
| `decision_logs.reporting.upload_size_limit_bytes` | int64 | `32768` | Max gzip-compressed payload size. |
| `decision_logs.reporting.min_delay_seconds` | int64 | `300` | Minimum wait between uploads. |
| `decision_logs.reporting.max_delay_seconds` | int64 | `600` | Maximum wait between uploads. |
| `decision_logs.reporting.trigger` | string | `periodic` | `periodic`, `immediate`, or `manual` — *"`manual` triggers are only possible when using OPA as a Go package"*. |

Four of these defaults are traps worth flagging for this project:

- **`buffer_size_limit_bytes` defaults to unlimited.** An unbounded in-memory decision buffer with
  an unreachable log sink is a slow memory leak that eventually kills the service — the same
  failure class as §4.3. Set it, or set `max_decisions_per_second`. Note they are mutually
  exclusive.
- **`min_delay_seconds` defaults to 300 and `max_delay_seconds` to 600.** Out of the box, a
  decision can sit in the buffer for up to ten minutes before upload, and is **lost on crash**.
  For an audit trail of security decisions that is too long; either lower it or use
  `trigger: immediate`.
- **`upload_size_limit_bytes` defaults to 32 KB**, and from the `nd_builtin_cache` documentation,
  *"decision logs that are larger than `upload_size_limit_bytes` will drop the `nd_builtin_cache`
  key from the log entry before uploading."* A PR diff input can be large; silently truncated or
  dropped audit records are worse than absent ones. Size this against your real input and monitor
  for drops.
- **`trigger: manual` only works when embedding OPA as a Go package** — which this project is
  doing. Combined with `immediate`, this gives precise control over when security decisions are
  durably recorded, rather than hoping the periodic flush beats the next crash.

The simplest useful configuration is one line:

```yaml
decision_logs:
  console: true
```

which emits every decision as structured JSON to stdout, where normal container log shipping picks
it up. **For a self-hosted single-binary product this is very likely the right default** — it
needs no extra infrastructure from the user, and it composes with whatever they already run.
Offer the remote `service` sink as opt-in for users who want a central audit store.

### 5.3 Masking and dropping

Masking is itself a Rego policy — the rule `mask` in package `system.log`. (The documentation
refers to it as `data.system.log.mask`; the configuration key `decision_logs.mask_decision`
expresses the same default in path form as `/system/log/mask`. They are the same rule.) It returns
JSON Pointers to erase, or structured operations to rewrite:

```rego
# erase
mask contains "/input/password" if { ... }

# rewrite
mask contains {"op": "upsert", "path": "/input/password", "value": "**REDACTED**"} if { ... }
```

Valid `op` values are `"remove"` and `"upsert"`. Pointers *"must be prefixed with `/input`,
`/result`, or `/nd_builtin_cache`"*. Erased and rewritten paths are recorded in the event's
`Erased` and `Masked` arrays — so the log states *that* something was redacted, which is what makes
a redacted audit log still trustworthy.

This matters more here than it might appear. A PR diff structure is **not obviously
non-sensitive**: file paths leak internal project structure, and if the input ever carries diff
*content* (not just structure) it can carry secrets a developer accidentally committed — which is
precisely the kind of thing a security policy would be written to detect. **A policy that detects
committed secrets must not log them.** Design the input to carry structure and metadata only
wherever possible, and mask whatever content remains.

Dropping suppresses whole events, via rules under `package system.log`:

```rego
package system.log

drop if {
    input.path == "kafka/allow"
}
```

*"The name of the drop rules by default is `drop` in the package `system.log`"*, configurable via
`drop_decision`. Use it to keep high-volume, low-value decisions out of the audit store — but
**never drop security decisions**, and consider making that structurally impossible by running
security and test-selection policy as separate SDK instances with separate decision-log
configuration, so a mistake in the test-selection drop rule cannot suppress a security audit
record.

### 5.4 Custom sinks

`v1/plugins/logs` declares a `Logger` interface (at
[`plugin.go#L39`](https://github.com/open-policy-agent/opa/blob/v1.21.2/v1/plugins/logs/plugin.go)),
and the plugin's upload path is `func (p *Plugin) Log(ctx context.Context, decision *server.Info) error`.
Combined with `sdk.Options.Plugins map[string]plugins.Factory`, this is the extension point for
writing decisions to your own store — Postgres, S3, an append-only audit table.

**Unverified:** I confirmed the `Logger` interface and the `Log` method exist at those locations but
did not read the interface's method set or write a working custom-plugin implementation. Treat
"implement `logs.Logger` and register via `sdk.Options.Plugins`" as the right shape and verify the
signatures before building on it.

For this project, a custom sink is probably worth it eventually: an audit trail that answers "why
was this PR blocked" needs to be **queryable by repository and PR**, which stdout JSON is not. But
start with `console: true`, ship, and add the sink when someone asks the question.

---
## 6. Prior Art

This project is an admission controller for pull requests. That is not an analogy — it is the same
control structure: an event arrives describing a proposed change, a policy engine evaluates the
change's *structure* against rules authored by someone other than the change's author, and the
result either admits the change, admits it with a warning, or routes it to a human. Kubernetes and
Terraform have run this pattern at scale for years and have made most of the available mistakes
already.

**This is the highest-value section in the document.** Several of the project's hardest
problems — graduated rollout, per-enforcement-point severity, the fail-open/fail-closed split,
multi-tenant policy isolation, and central policy distribution — are already solved here, and the
solutions are copyable. The headline finding: **the project's central thesis is not merely
defensible, it is the architecture every mature system in this space converged on**, and the one
open-source incumbent that gates pull requests is the one that has *not* got there.

### 6.1 OPA Gatekeeper

Gatekeeper is the OPA-based admission controller for Kubernetes, maintained in the **same GitHub
organization and under the same governance** as OPA (§8). Apache-2.0; a subproject of CNCF-graduated
OPA; current stable **v3.23.1** (2026-08-27) with roughly quarterly minors, shipping **OPA v1.17.1**
(there is a published [OPA version compatibility matrix](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/opa-versions.md)).

#### The two-level design: ConstraintTemplate and Constraint

**This is the most transferable piece of prior art available, and it maps onto the project's
multi-tenancy almost exactly.**

A **ConstraintTemplate** is a cluster-scoped CRD bundling two things: Rego (or CEL) source, and
*the OpenAPI schema of the parameters that source accepts*. Applying one causes Gatekeeper to
**generate a new CRD** whose kind is `spec.crd.spec.names.kind`. A **Constraint** is an instance of
that generated CRD. The docs frame it as *"fine-tune the behavior of a constraint, much like
arguments to a function"* ([howto](https://open-policy-agent.github.io/gatekeeper/website/docs/howto/)).

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        # Schema for the `parameters` field
        openAPIV3Schema:
          type: object
          properties:
            labels:
              type: array
              items:
                type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredlabels

        violation[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }
```

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: ns-must-have-gk
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Namespace"]
  parameters:
    labels: ["gatekeeper"]
```

Four mechanisms are doing work, and each answers a question this project has:

1. **`spec.crd.spec.names.kind` promotes the template to a first-class type.** Constraints are
   typed objects, not blobs in a config map.
2. **`openAPIV3Schema` type-checks the parameters, and this must fail loudly.** Structural schemas
   became mandatory in `v1` (Gatekeeper 3.6.0), and the API server now rejects a malformed
   Constraint before Gatekeeper sees it:
   ```
   The K8sRequiredLabels "ns-must-have-gk" is invalid: spec.parameters: Invalid value: "array":
   spec.parameters in body must be of type object: "array"
   ```
   **The documented regression this fixed is the single best argument in the corpus for the
   project's parameterisation design.** Under `v1beta1`'s non-structural schemas the bad Constraint
   was ingested successfully and then **silently did nothing** — namespaces were created without
   the required label and nobody was told. That is "parse, don't validate" applied to policy
   *configuration*: **a policy instance that fails to bind must be rejected at submit time, never
   degraded to a silent no-op.**
3. **`spec.match` separates *which objects* from *what rule*.** The Rego never decides scope; the
   constraint does, declaratively — so scope can be validated, listed and rendered in a UI without
   executing anything.
4. **`input.parameters` and `input.review.object` are a fixed input contract.** Rego-visible inputs
   are exactly `input.review`, `input.review.namespaceObject`, `input.parameters`, `data.lib.*`
   (from the template's `libs` field) and `data.inventory`. That fixed surface is what makes
   templates reusable — and, as §6.1's isolation discussion shows, what makes them containable.

The payoff for the project's thesis: **a platform engineer authors the template (the Rego, the hard
part, centrally, in the protected repository); a team lead authors the constraint (parameters and
scope, safely, without write access to the logic).** Both live outside the governed repository but
have different authors, different blast radius and different review requirements. This is strictly
better than "tenants write Rego", and it shrinks the §4 untrusted-code exposure to the much smaller
population who can write templates.

Note the convergence: Kubernetes' CEL `paramKind` (§7.1) is an independent design that arrived at
the same split.

The `Target` Go type, verified from
[constrainttemplate_types.go](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/core/templates/constrainttemplate_types.go):

```go
type Target struct {
	Target     string   `json:"target,omitempty"`
	Rego       string   `json:"rego,omitempty"`
	Libs       []string `json:"libs,omitempty"`
	Code       []Code   `json:"code,omitempty"`
	Operations []admissionv1.OperationType `json:"operations,omitempty"`
}

type Code struct {
	Engine string    `json:"engine,omitempty"`  // "Rego" | "K8sNativeValidation"
	Source *Anything `json:"source,omitempty"`
}
```

The newer `code[]` form carries an engine discriminator and lets a template opt into Rego v1 syntax
(Gatekeeper 3.19+; v0 remains the default). **One anti-pattern to avoid copying:** legacy
`spec.targets[].rego` silently wins over a Rego engine in `code[]`, and when several engines are
present *"only one is evaluated — CEL beats Rego with no fallback"*. Their own stated best practice
is to define logic in only one engine "to avoid confusion about which policy will be evaluated".
That is a design smell they are living with.

#### `enforcementAction`, and `scoped` — the direct answer to the project's hardest requirement

From [violations.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/violations.md),
verbatim: *"The supported enforcementActions are [`deny`, `dryrun`, `warn`] for constraints."*
Default `deny`; `--disable-enforcementaction-validation=true` disables validation against the list.

A **fourth value, `scoped`**, exists in the framework's Go constants
([apis.go](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/apis/constraints/apis.go)):

```go
const (
	Deny   EnforcementAction = "deny"
	Warn   EnforcementAction = "warn"
	Scoped EnforcementAction = "scoped"
)
```

(`dryrun` is absent from the Go constants — it is handled by the consuming layer, not the framework
core. Note also the doc comment on `Deny`: *"Errors encountered running validation are treated as
failing validation."* That matters below.)

**`dryrun` is a rollout primitive, not a logging mode.** *"When rolling out new constraints to
running clusters, the dry run functionality can be helpful as it enables constraints to be deployed
in the cluster without making actual changes... Cluster resources that are impacted by the dry run
constraint are surfaced as violations in the `status` field."* It evaluates against the **entire
existing population** and reports blast radius before enforcement — which only works because audit
exists (below).

`warn` (3.4+, k8s 1.19+) adds interactive feedback: the request succeeds and the user sees
```
Warning: [prod-repo-is-openpolicyagent] container <nginx> has an invalid image repo <nginx>, allowed repos are ["openpolicyagent"]
```

**Scoped enforcement is the single most relevant feature in this document to the project's
"fail closed on security, fail open on test selection" requirement.** From
[enforcement-points.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/enforcement-points.md),
the four enforcement points:

| Identifier | Meaning |
|---|---|
| `validation.gatekeeper.sh` | the admission webhook |
| `gator.gatekeeper.sh` | shift-left, via the gator CLI in CI |
| `audit.gatekeeper.sh` | periodic re-evaluation of existing resources |
| `vap.k8s.io` | Kubernetes-native ValidatingAdmissionPolicy (CEL only) |

```yaml
spec:
  enforcementAction: scoped
  scopedEnforcementActions:
  - action: warn
    enforcementPoints:
    - name: "validation.gatekeeper.sh"
  - action: deny
    enforcementPoints:
    - name: "gator.gatekeeper.sh"
```

Their own worked example is the project's use case almost verbatim: *"you want to deny violating
resources in shift-left testing, but do not want to block any resources admitted to clusters to
reduce impact for faulty rejections."* **One policy, different severity per enforcement point.**

And a second example is a latency argument worth reading twice: *"You are depending on
external-data or referential policies for validating resources. These type of validation may be
latency sensitive and may take longer to evaluate. To avoid such situation you may want to only use
`audit.gatekeeper.sh` enforcement point to not face any delay at admission time."* **A policy too
slow for the synchronous path can be demoted to the asynchronous one rather than dropped** — a
design option §2 does not otherwise suggest, and a good answer for an expensive policy under a
500 ms budget.

Two documented gotchas: `scopedEnforcementActions` is ignored unless `enforcementAction: scoped` is
also set; and *"the audit enforcement point is not included unless explicitly added... or if
`scopedEnforcementActions.enforcementPoints` is set to `"*"`"*. (**Unverified**: the feature is
present in the v3.17.x versioned docs and absent from v3.16.x, so "shipped in 3.17" is inferred
from docs presence rather than a release note.)

**Copy this structure, and note the crucial distinction it encodes.** `enforcementAction` governs
what happens when the policy **says no**. Fail-open/fail-closed (below) governs what happens when
the engine **cannot answer**. These are independent axes; conflating them is a design error. The
project should make severity a per-policy, per-enforcement-point declaration in policy metadata —
not two hard-coded Go call sites as §4.4 sketches.

#### Audit mode: the feature this project will wish it had on day one

Mechanism ([audit.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/audit.md)):
*"Audit performs periodic evaluations of existing resources against constraints, detecting
pre-existing misconfigurations."* By default it requests each resource from the API during each
cycle; `--audit-from-cache=true` reads the informer cache instead (which then requires sync).

Results land in three places by design: **Prometheus metrics**
(`gatekeeper_audit_last_run_time`, `gatekeeper_audit_last_run_end_time`, `gatekeeper_violations`);
**constraint `status`**; and **JSON audit logs** to stdout with `audit_id` tying a run together and
`event_type` ∈ `audit_started`, `violation_audited`, `constraint_audited`, `audit_finished`.

```yaml
status:
  auditTimestamp: "2019-05-11T01:46:13Z"
  enforced: true
  totalViolations: 2
  violations:
  - enforcementAction: deny
    group: ""
    version: v1
    kind: Namespace
    message: 'you must provide labels: {"gatekeeper"}'
    name: default
```

Written by the audit controller via `unstructured.SetNestedField`
([pkg/audit/manager.go](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/pkg/audit/manager.go)).

Flags and defaults:

| Flag | Default | Notes |
|---|---|---|
| `--audit-interval` | `60` (seconds) | `0` disables |
| `--constraint-violations-limit` | `20` | per constraint; `0` = store none |
| `--audit-chunk-size` | `500` | `0` = infinite; lower = less memory, more API calls |
| `--audit-from-cache` | `false` | requires replication |
| `--audit-match-kind-only` | `false` | only audit kinds named in constraints |
| `--enable-referential-rules` | `true` | see below |
| `--operation=audit` | — | removing it disables the audit operation |

**Why this matters enormously here.** An admission controller only sees *new* changes. It cannot
tell you how much of your existing estate violates a policy — and that is the first question anyone
asks before enabling one. The project's analogue: **evaluate each policy against the current
default branch of every governed repository, and against a corpus of recently merged PRs, on a
schedule, and publish the violation count.** That gives three things the admission path cannot:
rollout safety ("enabling this would have blocked 34 of your last 200 PRs" is the number that makes
a `dryrun` → `deny` decision possible), drift detection (policy or `main` changes and a repository
becomes non-compliant with no PR involved), and a reporting surface that exists whether or not
anyone is opening PRs.

**The bounded-resource lesson is the detail to copy.** *"If the number of current violations is
greater than this cap, the excess violations will not be reported (though they will still be
included in the `totalViolations` count). This is because Kubernetes has a cap on how large
individual API objects can grow, which makes unbounded growth a bad idea."* They keep the **count**
unbounded, truncate the **detail**, and say that they truncated. Never silently drop the fact that
there were more findings.

Audit is a **singleton** because concurrent writers contend on the same `status` fields — and even
with `--constraint-violations-limit=0`, `auditTimestamp` and `totalViolations` are still written, so
replicas still contend. If the project adopts status aggregation, that serialisation point is
inherent, not incidental.

A sharp correctness note with a direct consequence: *"the request's user's information, such as
`username`, `uid`, `groups`, `extra`, cannot be populated by Kubernetes for audit reviews and
therefore constraint templates that rely on `userInfo` are not auditable. It is up to the rego
author to handle the case where `userInfo` is unset."* **Any policy whose decision depends on the
actor cannot be re-evaluated later.** For this project that means PR-author-dependent policies
("bots may not change `/infra`") are not replayable against history — worth knowing before
promising an audit mode, and an argument for keeping actor-dependent and content-dependent rules in
separate policies.

#### Getting data not in the object under review

**(a) Sync → `data.inventory`.** From [sync.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/sync.md):
*"it is impossible to know if a label is unique across all pods and namespaces unless a
ConstraintTemplate has access to all other pods and namespaces."* Declared via `SyncSet` (3.15+,
alpha, recommended) or the singleton `Config` resource (3.6+); the **set union** of all SyncSets'
`gvks` and Config's `syncOnly` is synced.

```yaml
apiVersion: syncset.gatekeeper.sh/v1alpha1
kind: SyncSet
metadata:
  name: syncset-1
spec:
  gvks:
    - group: ""
      version: "v1"
      kind: "Namespace"
```

Access paths are rigidly structured: `data.inventory.cluster[<groupVersion>][<kind>][<name>]` and
`data.inventory.namespace[<namespace>][<groupVersion>][<kind>][<name>]`.

**The lesson is the allowlist, not the inventory.** Gatekeeper does not replicate the cluster — you
enumerate exactly which kinds get synced. That is the direct answer to §2.4's memory arithmetic:
per-tenant `data` inflates ~20× as JSON, so the correct pattern is an explicit allowlist of
reference data (CODEOWNERS, service catalogue, branch-protection state), not "load everything the
policy might want".

**(b) Referential constraints** — policies reading `data.inventory`. The honest caveat is in the
`--enable-referential-rules` help text (default `true`): *"Because referential constraints read
objects other than the object-under-test, they may be subject to race conditions. Users concerned
about this may want to disable referential rules."* A documented, opt-out-able consistency hole.
"Does another open PR already touch this file" is exactly a referential query with exactly this
race.

**(c) External data providers** (3.11+, beta) — and the
[motivation section](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/externaldata.md)
is the most important paragraph in the Gatekeeper corpus for this project's threat model, verbatim:

> A similar way to connect with an external data source can be done today using OPA's built-in
> `http.send` functionality. However, there are limitations to this approach.
> - Security concerns due to:
>   - **if template authors are not trusted, it will potentially give template authors access to the in-cluster network.**
>   - if template authors are trusted, authors will need to be careful on how rego is written to avoid injection attacks.

Their fix was not to sandbox `http.send` but to **delete it and replace it with a mediated
protocol** — a `Provider` CRD naming an allowed URL, timeout and CA bundle, plus a batching builtin:

```yaml
apiVersion: externaldata.gatekeeper.sh/v1beta1
kind: Provider
metadata:
  name: my-provider
spec:
  url: https://my-provider.my-namespace:8090/validate
  timeout: 1
  caBundle: <caBundle>
```

```rego
my_list := [img | img = input.review.object.spec.template.spec.containers[_].image]
response := external_data({"provider": "my-provider", "keys": my_list})
```

Responses are a triple-array per key — `[["my-key", "my-value", ""], ["bad-key", "", "error message"]]`
— value and error in-band, per key. Stated benefits: *"Restricting which hosts a user can access"*
and *"Providing an interface for making requests, which allows Gatekeeper to better handle things
like escaping strings."*

**And they shipped the enforcement, which independently confirms §4.2.** From
[customize-startup.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/customize-startup.md):
*"Starting with v3.8.0, Gatekeeper disables the `http.send` built-in function by default."*
Confirmed in `main.go`:

```go
flag.Var(disabledBuiltins, "disable-opa-builtin", "disable opa built-in function, this flag can be declared more than once.")
...
args := []rego.Arg{rego.Tracing(false), rego.DisableBuiltins(disabledBuiltins.ToSlice()...)}
```

**Take this wholesale.** This project runs partially-untrusted tenant Rego in a service holding a
GitHub App private key inside its own network. Strip `http.send` at compile time (§4.2), and if
tenants need external lookups, mediate them through a narrow provider interface you control — which
also gives you the single place to apply timeouts, caching and host allowlisting that `http.send`
cannot.

#### Multi-tenant Rego isolation: two framework internals worth stealing

**`regorewriter` — namespace isolation.**
[pkg/regorewriter](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/regorewriter/regorewriter.go)
*"rewrites rego code by updating library package paths by prepending a prefix and updating
references to library code accordingly"*, with `allowedLibPrefixes` and `allowedExterns`. The Rego
driver applies it per template:

```go
rr, err := regorewriter.New(regorewriter.NewPackagePrefixer(templateLibPrefix), []string{libRoot}, externs)
```

and the allowed externs are a closed allowlist (`args.go`):

```go
var validDataFields = map[string]bool{"inventory": true}
```

**Tenant Rego can reference `data.inventory` and nothing else under `data`.** That is the
multi-tenant isolation primitive, and `Externs(...)` + `DisableBuiltins(...)` together form a
compile-time sandbox — a materially stronger position than §4.2 alone. (OPA bundle `roots`, §3.1,
achieves the same namespace containment by a different route; either works, and belt-and-braces is
reasonable here.)

**The hook-module trick — one evaluation per template, not per constraint.** From
[drivers/rego/rego.go](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/client/drivers/rego/rego.go),
the framework injects:

```rego
package hooks

violation[response] {
  key := input.constraints[_]
  inp := {
    "review": input.review,
    "parameters": data.constraints[key.kind][key.name],
  }
  data.template.violation[r] with input as inp
  response := {"key": key, "details": object.get(r, "details", {}), "msg": r.msg}
}
```

Every template's Rego is rewritten to package `template`, constraint parameters live in the store at
`data.constraints[kind][name]`, and **all constraints of a template are evaluated in one query**
that iterates them and re-binds `input` via `with`. One `*ast.Compiler` per (target, kind). **That
is the performance architecture for "many parameterised instances of few policies" — which is
exactly this project's shape** — and it removes boilerplate from every template.

**But note the gap, and do better:** the driver calls `rego.New(args...)` then `r.Eval(ctx)` on
**every evaluation** and never uses `rego.PreparedEvalQuery`. It reuses the compiler but re-plans
the query each time. Against a 500 ms budget, use `PrepareForEval` and cache the prepared query
keyed on bundle revision (§1.2).

#### Fail-open vs fail-closed: the most valuable finding in this document

**Gatekeeper's shipped default is fail-open, and the asymmetry in *which* webhook fails closed is
the thing to copy.** Verified from the shipped manifest
([deploy/gatekeeper.yaml](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/deploy/gatekeeper.yaml))
rather than from prose:

| Webhook | `failurePolicy` | `timeoutSeconds` |
|---|---|---|
| `validation.gatekeeper.sh` (policy) | **`Ignore`** | 3 |
| `check-ignore-label.gatekeeper.sh` (exemption guard) | **`Fail`** | 3 |
| `mutation.gatekeeper.sh` | `Ignore` | 1 |

From [failing-closed.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/failing-closed.md):
*"Currently Gatekeeper is defaulting to using `failurePolicy: Ignore`... The impact of this is that
when the webhook is down, or otherwise unreachable, constraints will not be enforced. **Audit is
expected to pick up any slack in enforcement** by highlighting invalid resources that made it into
the cluster."*

**The policy webhook fails open. The one webhook that fails closed is the one guarding the
mechanism by which policy can be bypassed.** Here is why that is not an accident. Namespace
exemption works by labelling a namespace `admission.gatekeeper.sh/ignore`, and from
[exempt-namespaces.md](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/website/docs/exempt-namespaces.md),
verbatim:

> This step is necessary because otherwise **the permission to modify a namespace would be
> equivalent to the permission to exempt everything in that namespace from policy checks.** This
> way a user must explicitly have permissions to configure the Gatekeeper pod before they can add
> exemptions.

The guard webhook enforces it
([pkg/webhook/namespacelabel.go](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/pkg/webhook/namespacelabel.go)):

```go
// +kubebuilder:webhook:verbs=CREATE;UPDATE,path=/v1/admitlabel,mutating=false,failurePolicy=fail,...
const ignoreLabel = "admission.gatekeeper.sh/ignore"
...
if exemptNamespace[obj.GetName()] || matchesPrefix(...) || matchesSuffix(...) {
    return admission.Allowed(...)
}
for label := range obj.GetLabels() {
    if label == ignoreLabel {
        return admission.Denied(fmt.Sprintf("Only exempt namespace can have the %s label", ignoreLabel))
    }
}
```

Exemption must be pre-authorised by `--exempt-namespace` on the **Gatekeeper process** — a flag
only whoever deploys Gatekeeper can set. **That sentence, with "namespace" replaced by "repository",
is this project's core security property.** The governed party cannot grant itself an exemption, and
the control preventing self-exemption is the one that fails closed.

**A second, finer nuance that §4.4 should adopt directly: unavailability fails open, but
*evaluation errors* fail closed.** The `Deny` constant's doc comment says errors are treated as
failing validation, and the webhook confirms it
([pkg/webhook/policy.go](https://raw.githubusercontent.com/open-policy-agent/gatekeeper/master/pkg/webhook/policy.go)):

```go
resp, err := h.reviewRequest(ctx, &req)
if err != nil {
    h.log.Error(err, "error executing query")
    requestResponse = errorResponse
    return admission.Errored(http.StatusInternalServerError, err)
}
```

`admission.Errored` in controller-runtime returns `Allowed: false`. **So a Rego runtime error denies
the request**; `failurePolicy: Ignore` only covers the case where the API server cannot get a
response at all. Three distinct decisions, each separately argued:

| Condition | Outcome | Rationale |
|---|---|---|
| Cannot reach the engine | **allow** | availability |
| Engine ran and errored | **deny** | correctness |
| Someone attempts self-exemption | **deny, always** | integrity |

That three-way split is sharper than the two-way split in §4.4 and should replace it. Mapped onto
this project: a bundle that failed to download → fall back to the persisted last-known-good
(§3.4); a policy that raised `eval_builtin_error` → block and report; a tenant trying to disable
enforcement for their own repository → always refuse, and make that path fail closed.

**The documented risks of failing closed**, which read as a warning to any self-hosted admission
controller:

- **Admission deadlock.** *"Imagine you delete every `Node` in your cluster. This will kill all
  running Gatekeeper servers... Because a request to add a `Node` is subject to admission
  validation, it cannot succeed until the webhook can serve. The webhook cannot serve until a
  `Node` is added. This circular dependency will need to be broken."* Mitigation: delete the
  `ValidatingWebhookConfiguration`, which always works because *"Kubernetes does not make requests
  to edit webhook configurations subject to admission webhooks."* **The emergency-recovery path is
  deliberately outside the thing being enforced.**
- **A gotcha on that mitigation:** *"If the existence of the webhook resource is enforced by some
  external process (such as an operator), that may interfere with the emergency recovery process."*
  A GitOps reconciler will undo your break-glass.
- **Control-plane availability coupling**: *"the availability of K8s's control plane becomes subject
  to the availability of the webhook."*
- **Exemption-based mitigation is unreliable**: *"It can be hard to say for certain that all
  critical resources have been exempted because dependencies can be non-obvious... Did you know
  that a `ConfigMap` is used as the locking resource for some Kubernetes leader elections?"*
- **Replicas do not help within one failure domain** — same node, host, rack, switch, power supply.

And the closer, which is about this project: *"In a nutshell it's because it's a webhook, and
because it's self-hosted... **Any self-hosted admission controller would be subject to these same
concerns.**"*

**Three consequences for the design:**

1. **The circular-dependency failure is reproducible here.** If the control plane's own repository
   is governed by the control plane, a policy change that breaks evaluation cannot be fixed by a
   PR, because the PR cannot pass. **Exempt the control plane's own repository and the policy
   repository from enforcement deliberately, from day one, and build a break-glass path outside the
   enforced system before you need it** — and make sure your own deployment automation will not
   immediately undo it.
2. **Fail-closed is a per-policy decision, not a per-system one.** Blocking every PR across every
   repository because one bundle failed to verify is an outage, not a security win. Gatekeeper's
   answer — fail open on availability, and let *audit* catch what slipped through — is a genuinely
   better design than blanket fail-closed, and it only works because audit exists.
3. **A short timeout is part of the contract.** `timeoutSeconds: 3` against §2.1's ~36 µs median is
   three orders of magnitude of headroom while still bounding damage.

**One verified gap, which is this project's to close.** There is **no per-evaluation deadline** in
either the Rego driver
([drivers/rego/driver.go](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/client/drivers/rego/driver.go)
— no `WithTimeout`, no `Deadline`) or the validation handler. The only bounds are the API server's
`timeoutSeconds: 3` and `--max-serving-threads` (default `-1` = `GOMAXPROCS`), which *"caps the
number of concurrent goroutines that are calling out to policy evaluation at any one time"*.
Cancellation does propagate via the inbound request context, so the webhook path is not unbounded
in practice — but **nothing bounds a pathological comprehension from a hostile template author
beyond the caller hanging up.** This is §4.3's gap, confirmed independently in the most mature
system in the space. Set your own `context.WithTimeout`, and treat it as a policy-ingest concern
too.

#### Mutation and Expansion (brief)

**Mutation** (3.10+, stable) — *"allows Gatekeeper to modify Kubernetes resources at request
time."* Deliberately *not* ConstraintTemplate-shaped: four mutator CRDs (`AssignMetadata`,
`Assign`, `ModifySet`, `AssignImage`), each split into *extent of changes* (match), *intent*
(`location` + value) and *conditional*. **Rego is not available for mutation at all** — part of why
external data exists.

**ExpansionTemplate** (3.10+ alpha, 3.13+ beta, on by default) — solves "a Deployment is admitted,
but the Pod it will create is what violates policy". Gatekeeper synthesises *"temporary, fake
resources"* from `templateSource` and evaluates Pod-targeting constraints against them, with an
optional `enforcementAction` override applied **only to expanded resources**:

```yaml
apiVersion: expansion.gatekeeper.sh/v1alpha1
kind: ExpansionTemplate
spec:
  applyTo:
    - groups: ["apps"]
      kinds: ["DaemonSet", "Deployment", "ReplicaSet", "StatefulSet"]
      versions: ["v1"]
  templateSource: "spec.template"
  enforcementAction: "warn"
  generatedGVK:
    kind: "Pod"
    group: ""
    version: "v1"
```

**The transferable idea:** evaluate policy against *what the change will cause*, not only what it
literally says — and give the derived evaluation a **softer default severity, because the
derivation itself may be wrong.** If the project ever expands a diff into "the resulting file state"
or "the dependencies this will pull in", copy the severity downgrade.

#### Can the constraint framework be reused outside Kubernetes? Yes — and there is a production proof

This is the question with the largest practical consequence, and the answer is more interesting than
a flat no.

**The extension point is one interface**
([pkg/handler/handler.go](https://raw.githubusercontent.com/open-policy-agent/frameworks/master/constraint/pkg/handler/handler.go)):

```go
type TargetHandler interface {
	crds.MatchSchemaProvider          // MatchSchema() apiextensions.JSONSchemaProps
	GetName() string                  // must match ^[a-zA-Z][a-zA-Z0-9.]*$
	ProcessData(data interface{}) (handle bool, key []string, inventoryFormat interface{}, err error)
	HandleReview(object interface{}) (handle bool, review interface{}, err error)
	ValidateConstraint(constraint *unstructured.Unstructured) error
	ToMatcher(constraint *unstructured.Unstructured) (constraints.Matcher, error)
}
```

with the client API:

```go
func NewClient(opts ...Opt) (*Client, error)
func Targets(ts ...handler.TargetHandler) Opt
func Driver(d drivers.Driver) Opt
func EnforcementPoints(eps ...string) Opt

func (c *Client) AddTemplate(ctx, templ *templates.ConstraintTemplate) (*types.Responses, error)
func (c *Client) AddConstraint(ctx, constraint *unstructured.Unstructured) (*types.Responses, error)
func (c *Client) AddData(ctx, data interface{}) (*types.Responses, error)
func (c *Client) Review(ctx, obj interface{}, opts ...reviews.ReviewOpt) (*types.Responses, error)
```

The repo README frames reuse as the point: *"The OPA Frameworks repository defines opinionated APIs
for policy that are less flexible than the OPA API but are well-suited to particular classes of use
cases... Frameworks are to OPA as Rails is to Ruby."* The `Client` doc comment is candid about
concurrency: *"Threadsafe. Does not support concurrent mutation operations... this problem can only
safely be handled by the caller."*

**It does hard-depend on Kubernetes *type packages*, but not on a cluster.** Direct requires in
`constraint/go.mod` include `k8s.io/api`, `k8s.io/apiextensions-apiserver`, `k8s.io/apimachinery`,
`k8s.io/client-go` and `sigs.k8s.io/controller-runtime`. Concretely, you cannot escape
`*unstructured.Unstructured` as the representation of a Constraint, nor
`apiextensions.JSONSchemaProps` as the representation of your match schema. So "usable outside
Kubernetes" is **true** in the sense that matters — no API server, no CRDs, no cluster — and
**false** in the sense that you link a large slice of the Kubernetes type ecosystem into a binary
whose selling point is being a single self-hostable artifact.

**The production proof: `GoogleCloudPlatform/config-validator`.** Among the 189 packages importing
`pkg/client` are `GoogleCloudPlatform/config-validator` and `Azure/draft`. config-validator
(Apache-2.0) ships **two non-Kubernetes target handlers** — `pkg/gcptarget` and, decisively,
**`pkg/tftarget`**, a target for Terraform resource *changes*:

```go
const Name = "validation.resourcechange.terraform.cloud.google.com"

func (g *TFTarget) HandleReview(obj interface{}) (bool, interface{}, error) {
	switch resource := obj.(type) {
	case map[string]interface{}:
		if _, found, err := unstructured.NestedString(resource, "name"); !found || err != nil { return false, nil, err }
		if _, found, err := unstructured.NestedString(resource, "address"); !found || err != nil { return false, nil, err }
		if _, found, err := unstructured.NestedMap(resource, "change"); !found || err != nil { return false, nil, err }
		if _, found, err := unstructured.NestedString(resource, "type"); !found || err != nil { return false, nil, err }
		return true, resource, nil
	}
	return false, nil, nil
}

// ProcessData implements handler.TargetHandler
func (g *TFTarget) ProcessData(obj interface{}) (bool, []string, interface{}, error) {
	return false, nil, nil, errors.Errorf("storing data for referential constraint eval is not supported at this time.")
}
```

What this proves, point by point, maps onto this project almost line for line:

1. **The reviewed object is a Terraform plan `resource_changes` entry** — a *diff record*, not a
   Kubernetes object. The framework genuinely does not care.
2. **The target owns `spec.match` entirely.** `MatchSchema()` returns a schema with
   `addresses`/`excludedAddresses` instead of `kinds`/`namespaces`, and the matcher globs on the
   resource address with `.` as separator (using `gobwas/glob`, supporting `*` and `**`). **For a
   PR service this is exactly where file-path globs go** — "this rule applies to `src/auth/**`" as a
   first-class, schema-validated field rather than a convention.
3. **`ProcessData` returning an error is a legitimate choice.** You can decline referential data
   entirely and keep the rest of the model.
4. **Wiring is ordinary Go:**

```go
driver, _ := rego.New(rego.Tracing(false))
cfClient, _ := cfclient.NewClient(cfclient.Targets(tftarget.New()), cfclient.Driver(driver))
cfClient.AddTemplate(ctx, template)
cfClient.AddConstraint(ctx, constraint)
responses, _ := v.tfCFClient.Review(ctx, inputResource)
```

5. **And it ships in a Google product.** `gcloud beta terraform vet` validates Terraform plans
   against `constraints.gatekeeper.sh/v1alpha1` constraints held in a **separate Git repository**
   ([docs](https://docs.cloud.google.com/docs/terraform/policy-validation/validate-policies)):

```bash
terraform plan -out=tfplan.tfplan
terraform show -json ./tfplan.tfplan > ./tfplan.json
git clone POLICY_LIBRARY_REPO POLICY_LIBRARY_DIR
gcloud beta terraform vet tfplan.json --policy-library=POLICY_LIBRARY_DIR
```

Exit 0 = clean, exit 2 = violations as JSON with constraint name, message, resource, severity,
metadata. **That is the Gatekeeper template/constraint model, evaluating a diff, with policy in an
external repository, outside Kubernetes, in production.** It is the closest existing thing to this
project.

**The caveats, which decide the recommendation:**

- **The `constraint` submodule has never been released.** Zero `constraint/`-prefixed tags, and the
  Go proxy's version list is **empty** — pseudo-versions only. Gatekeeper master pins
  `v0.0.0-20260616163050-e1eaa1bf6d62`. **No semver, no API stability promise.**
- **config-validator is pinned to a 2023 pseudo-version** and `opa v0.54.0`, against the framework's
  current `opa v1.21.1`. The interface has drifted (`HandleViolation` in tftarget is no longer in
  the interface). Nobody tracks master downstream.
- **Mandatory Kubernetes types in your own API surface**, as above.
- `NewClient` now errors with *"must specify at least one enforcement point with
  client.EnforcementPoints"*.

**Verdict: reimplement the pattern, do not import the library — but read `tftarget` and
`gcv/validator.go` first.** They are a ~200-line worked example of this exact problem, and
`MatchSchema()`/`ToMatcher()` is the seam you want. Reimplementing gets you the two-level split, the
target-owned match schema, and the reject-at-submit-time property for a few hundred lines of Go,
without `unstructured.Unstructured` in your domain model, without controller-runtime, and without
depending on an untagged module that Gatekeeper itself tracks by commit hash.

#### `gator`: the CI story, and a model for the project's own CLI

`gator` (3.11+, beta; `go install github.com/open-policy-agent/gatekeeper/v3/cmd/gator@master`,
or `brew install gator`) is the shift-left face of the same engine
([docs](https://open-policy-agent.github.io/gatekeeper/website/docs/gator/)).

- **`gator test`** — evaluates objects against templates and constraints. Inputs via repeatable
  `--filename`/`-f`, repeatable `--image`/`-i` (**OCI artifacts**, `--plain-http` for local
  registries), and stdin. Exit `0` clean; `1` for `deny`-level violations; `dryrun`/`warn`
  violations print but **do not** change the exit code — *"meant to make the exit code of `1`
  consistent with rejection of the object by Gatekeeper's webhook."* `--output=json|yaml`,
  `--deny-only` (3.19+).
  ```
  cat my-manifest.yaml | gator test --filename=template-and-constraints/
  ```
  **Do not copy one thing:** *"An error during evaluation, for example a failure to read a file,
  will result in a `1` exit status"* — **crash and policy-denial are indistinguishable by exit
  code.** Conftest has the same defect (§6.2). Given §4.4, the project should use distinct codes.
- **`gator verify`** — a three-level policy *unit test* framework: Suite
  (`kind: Suite`, `apiVersion: test.gatekeeper.sh/v1alpha1`) → Test (one template + one constraint)
  → Case (one object + assertions). The assertion language is good:
  ```yaml
    - name: both-disallowed
      object: samples/repo-must-be-openpolicyagent/disallowed_both.yaml
      assertions:
      - violations: 2
      - message: initContainer
        violations: 1
      - message: container
        violations: 1
  ```
  `violations` is `yes`/`no`/a non-negative integer; `message` is a case-sensitive regex;
  `inventory:` supplies `data.inventory` for referential tests. **If end users author policies, you
  owe them this.** Note `violations: no` plus a `message` is the **denial-test primitive** — assert
  that a *specific* violation is absent.
- **`gator bench`** — `--engine rego|cel|all`, `--iterations` (default 1000, *"Use ≥1000 for
  reliable P99 percentiles"*), `--concurrency`, `--save`/`--compare` against a baseline,
  `--threshold` (default 10%) and `--min-threshold` for **CI regression gating**. With an honest
  caveat: *"`gator bench` measures compute-only policy evaluation latency, which does not include
  network round-trip time, TLS overhead, or Kubernetes API server processing... Use these metrics
  for relative comparisons between policy versions, not as absolute production latency
  predictions."* **A latency-regression gate on tenant policies in CI is what a sub-500 ms SLO
  actually requires over time, and here is a working design.**
- **Policy distribution** is OCI artifacts built with the **ORAS CLI**, with no proprietary schema —
  *"`gator` does not enforce any file schema in the artifacts"*:
  ```shell
  oras push localhost:5000/gator/policy-bundle:v1 ./constraints/:application/vnd.oci.image.layer.v1.tar+gzip \
    ./template_library/:application/vnd.oci.image.layer.v1.tar+gzip
  ```
- **`gator policy`** (3.22+, alpha) — a *"brew-inspired"* package manager over the
  gatekeeper-library: `search`, `list`, `install`, `uninstall`, `update`, `upgrade`, with bundles
  (`gator policy install --bundle pod-security-baseline`) and `--dry-run`. A curated central
  catalogue with safe-rollout defaults (install at warn enforcement) — relevant if the project
  plans a policy marketplace.

**Ship an equivalent CLI and treat it as a first-class deliverable.** Two distinct needs: a *policy
author* needs `gator verify` (run the policy's own tests in the policy repository's CI, so a bad
policy never reaches a bundle — §3.6 step 2); a *governed developer* needs `gator test` (pipe a
local diff in and learn whether it would be blocked **before** opening the PR). Without the second,
"policy you cannot edit" reads as "policy you cannot predict either", which is the difference
between a guardrail and an obstacle. And because `gator test` accepts `--image=`, the CLI pulls the
**same signed artifact the server enforces**, so local and server verdicts cannot drift. Copy that.

### 6.2 Conftest

[Conftest](https://www.conftest.dev/) is *"a utility to help you write tests against structured
configuration data"*, and is a **CLI wrapping OPA as a Go library** (`policy/engine.go` imports
`opa/v1/{ast,bundle,loader,rego,storage,topdown}`, pinned to `opa v1.21.1`). Apache-2.0, an official
OPA subproject since July 2020, current release **v0.71.1** (2026-10-06) — **still 0.x after 7.5
years, with no API stability promise on `policy.Engine`**, and bursty releases with 2.7–3.6 month
gaps.

Its value over raw OPA is three things: a parser fleet, a rule-naming convention, and a CI shape.

**23 parsers**: `codeowners`, `cue`, `cyclonedx`, `dockerfile`, `edn`, `groovy`, `hcl1`, `hcl2`,
`hocon`, `ignore`, `ini`, `json`, `jsonc`, `jsonnet`, `nginx`, `properties`, `spdx`, `textproto`,
`toml`, `vcl`, `xml`, `yaml`, `dotenv`. (A verified discrepancy: `parser.Parsers()` returns only 22
— it omits `jsonc`, so `jsonc` is absent from `--parser`'s advertised values even though
`New("jsonc")` works.) Auto-detection falls back to *using the file extension verbatim as the
parser name*, and stdin is **always** parsed as YAML.

**The rule-naming spec is two regexes** (`policy/engine.go`, verified at v0.71.1):

```go
warningRegex = regexp.MustCompile("^warn(_[a-zA-Z0-9]+)*$")
failureRegex = regexp.MustCompile("^(deny|violation)(_[a-zA-Z0-9]+)*$")
```

Matches `deny`, `violation`, `warn`, `deny_foo`, `deny_foo_bar`. Does **not** match `Deny`,
`denyFoo`, `deny_`, `deny_my-rule`, `deny_my.rule`. `violation` is now **identical** to `deny` —
same regex, same code path; the docs' claim that only `violation` supports structured data is stale.
⚠️ **A non-matching rule name is silently skipped** — `deny_my-rule` is never evaluated and the file
passes with no warning.

Deny rules must be partial set rules; `deny if { ... }` without `contains` is rejected at load time.
Return values are handled in a two-case switch over `string` and `map[string]any` (requiring a
`msg` string key; `_loc` is also reserved; everything else becomes metadata) — **with no `default`
branch**. ⚠️ Anything else (number, bool, array, set, null) is **silently discarded**, and because
later arithmetic counts unproduced results as successes, **`deny contains 42` is a silent PASS.**
For an admission controller a malformed rule must fail closed; the project's version needs a
`default:` that errors.

**Exit codes** (`output/result.go` + [options](https://www.conftest.dev/options/)): `0` pass; `1`
when `deny`/`violation` fires; warning-only is `0` in default mode; **`1` for errors too** (bad
flag, unparseable file, Rego compile error). With `--fail-on-warn`: `0` clean / `1` warn-only / `2`
any failure. ⚠️ **In default mode "a policy denied" and "Conftest crashed" are both exit 1.**

Policy distribution is the strongest area. `conftest pull` / `conftest push` / `conftest test
--update`:

```console
conftest pull oci://opa.azurecr.io/test
conftest pull git::https://github.com/<Org>/<Repo>.git//sub/folder
conftest push opa.azurecr.io/test
conftest test --update oci://registry.example.com/team/policies:latest deployment.yaml
```

Fetching uses `hashicorp/go-getter` as the outer framework (file/git/hg/s3/gcs/http) plus a
Conftest-authored `OCIGetter` on the `oci` scheme using `oras-go/v2`; auth comes from the Docker
credential store. **Crucially, the layer media types are OPA's own:**

```go
openPolicyAgentPolicyLayerMediaType = "application/vnd.cncf.openpolicyagent.policy.layer.v1+rego"
openPolicyAgentDataLayerMediaType   = "application/vnd.cncf.openpolicyagent.data.layer.v1+json"
```

so a Conftest-pushed artifact **is an interoperable OPA bundle**, not a proprietary format — the
docs confirm Conftest *"reuses the Bundle defined by Open Policy Agent."*

⚠️ **Sharp edge: OCI auto-detection uses a hardcoded registry allowlist**
(`downloader/oci_detector.go`): `azurecr.io`, `gcr.io`, `registry.gitlab.com`, `pkg.dev`, ECR, and
`^quay.io`. **`ghcr.io` is not on it**, nor Docker Hub, Harbor, Artifactory, or anything
self-hosted — which is the unexplained reason the docs say "always include the `oci://` scheme for
private or on-premises registries".

#### Should the project reuse Conftest? No — and the reasons are stronger than expected

The complete `policy` package surface:

```go
type CompilerOptions struct { Strict bool; RegoVersion string; Capabilities *ast.Capabilities }
type Engine struct { /* all fields unexported */ }

func Load(policyPaths []string, opts CompilerOptions) (*Engine, error)
func LoadWithData(policyPaths []string, dataPaths []string, opts CompilerOptions) (*Engine, error)
func (e *Engine) Check(ctx context.Context, configs map[string]interface{}, namespace string) (output.CheckResults, error)
func (e *Engine) CheckCombined(ctx context.Context, configs map[string]any, namespace string) (output.CheckResult, error)
func (e *Engine) Compiler() *ast.Compiler
func (e *Engine) Runtime() *ast.Term
// ... Store, Modules, Namespaces, Documents, Policies, EnableTracing, EnableInterQueryCache, ShowBuiltinErrors
```

Five findings decide it:

1. **`Engine` can only be constructed from filesystem paths.** All fields unexported; the only
   constructors take `[]string` paths through `loader.NewFileLoader()`. There is **no** exported way
   to supply an `*ast.Compiler`, an `*ast.Module`, a bundle, an `fs.FS`, or a Rego string;
   `Compiler()` is a getter, not a setter. **Policy from a central external source must be
   materialised to local disk first. This alone is disqualifying for the project's thesis.**
2. **`Engine.Check` is not concurrency-safe and mutates shared state per call.** `addFileInfo`
   opens a **write transaction on the shared store** and writes `storage.Path{"conftest","file"}` on
   every check, and lazily initialises `e.store = inmem.New()` **without a mutex**. One `*Engine`
   serving concurrent PRs is a real race.
3. **No prepared queries.** `Engine.query` does `rego.New(options...)` then `.Eval(ctx)` —
   `PrepareForEval` appears **zero** times in `policy/engine.go` — and `check` issues **two** such
   queries *per distinct rule name*. Thirty `deny_*` rules means 60 fresh plan-and-eval cycles per
   input document. Wrong shape for a sub-500 ms budget. (Reasoned from the code path, not
   benchmarked.)
4. **`Engine.Runtime()` injects the entire process environment into tenant-visible Rego**, and
   `query` passes it unconditionally via `rego.Runtime(e.Runtime())`:
   ```go
   func (e *Engine) Runtime() *ast.Term {
       env := ast.NewObject()
       for _, pair := range os.Environ() { ... }
       obj.Insert(ast.StringTerm("env"), ast.NewTerm(env))
   ```
   Any tenant policy can write `opa.runtime().env.GITHUB_APP_PRIVATE_KEY` and exfiltrate it in a
   `deny` message. **There is no exported way to turn this off.** For a service holding a GitHub App
   key and running tenant Rego this is a **disqualifying security defect**.
5. **Dependency weight: 36 direct, 129 indirect** — including `cuelang.org/go`,
   `moby/buildkit v0.33.1`, `bufbuild/protocompile`, `CycloneDX/cyclonedx-go`, `spdx/tools-golang`,
   and transitively the AWS SDK, GCS and Mercurial. For a single self-hostable binary that is CUE's
   compiler, BuildKit's Dockerfile frontend, protobuf's compiler, three cloud SDKs and an SBOM
   library — to evaluate Rego against a struct already in memory.

Also: `runner.TestRunner.Run` takes file paths and **re-compiles the entire policy set on every
call**; commands live under `internal/`; `os.Exit` and `logger.Fatalf` appear inside command
construction; and the library packages **read global viper state** (`parser.New` reads
`proto-file-dirs`, `internal/registry.SetupClient` reads `tls`), so `parser.New("textproto")`
behaves differently depending on process-global state you never passed it.

One good-news verification: `builtins/parse_config.go` registers `parse_config`,
`parse_config_file` and `parse_combined_config_files` globally in an `init()`, and two of them
`os.ReadFile` a path joined relative to the policy file's directory **with no traversal guard** — an
arbitrary-file-read primitive for untrusted authors. Only `internal/commands/default.go`
blank-imports that package, so importing `policy` or `parser` does **not** pull it in. **Never add
`_ ".../conftest/builtins"` yourself.**

Nor as a subprocess: Conftest's model is *files on disk*, while this project's input is an in-memory
diff structure. And it has nothing to say about the project's actual hard problems — no decision
log, no bundle signature verification, no long-running service, no multi-tenancy.

**Four things to copy instead:**

1. **The `deny`/`violation`/`warn` convention with optional `_suffix`** — multi-reason output for
   free, mapping cleanly onto §6.1's three-state enforcement, and familiar to anyone who has
   written Conftest or Gatekeeper policy. **With two corrections:** add a `default:` branch that
   **errors** on a non-string/non-map value, and validate strictly that `msg` is present and a
   string.
2. **The graduated exit codes** (0 clean / 1 warnings / 2 failures) — `dryrun`/`warn`/`deny`
   expressed as a CLI contract. **But give errors their own code**, which neither Conftest nor
   `gator test` does.
3. **`--combine`** — *"combines files into one `input` data structure. The structure is an `array`
   where each element is a `map` with two keys: a `path` key with the relative file path of the file
   being evaluated and a `contents` key containing the actual document."* That is almost exactly
   the shape a PR diff should take: a list of per-file records with a path and a payload, evaluated
   as one document so a policy can reason *across* files. **A validated input-shape precedent.**
4. **`--output github`** (and `sarif`) — if the project writes inline PR annotations, match the
   format rather than inventing one.

### 6.3 Atlantis — the closest existing system to this project

`runatlantis/atlantis`: Go, ~9.3k stars, actively maintained, *"Terraform Pull Request
Automation"* — self-hosted, consumes GitHub PR webhooks, comments decisions back. *"Atlantis
supports running server-side conftest policies against the plan output"*
([policy-checking.md](https://raw.githubusercontent.com/runatlantis/atlantis/main/runatlantis.io/docs/policy-checking.md)).

**Policy config lives in the server-side repo config, not the governed repo** (`--repo-config`,
`--repo-config-json`, or `ATLANTIS_REPO_CONFIG_JSON`):

```yaml
policies:
  owners:
    users:
      - nishkrishnan
  policy_sets:
    - name: deny_null_resource
      path: <CODE_DIRECTORY>/policies/deny_null_resource/
      source: local
    - name: deny_local_exec
      path: <CODE_DIRECTORY>/policies/deny_local_exec/
      source: local
      approve_count: 2
      owners:
        users:
          - pseudomorph
```

Verified semantics: `source` — *"Tells atlantis where to fetch the policies from. Currently you can
only host policies locally by using `local`."* `owners` — who may approve a specific policy set.
`approve_count` — approvals needed to bypass. `prevent_self_approve` — whether the PR author may
approve. `sticky_policy_approvals` + `policy_item_regex` — whether approvals survive re-plans.

**The approval model is this project's "route to human review", already built, and its default is
the right one:** *"By default, any plans following the approval will discard all policy approvals
and prompt again for them."* **A human approval binds to a specific evaluated artifact, not to the
PR.** New commit, new decision, approval void — with `sticky_policy_approvals` as the opt-out, gated
by a regex that extracts comparable items so approval survives only while no *new* findings appear.
The project will face this question on day one; Atlantis shipped both answers, and
`prevent_self_approve` plus per-policy-set `owners` is the privilege separation to copy.

**Two findings to take seriously.**

First, this warning on `allow_custom_workflows` (default `false`) is the project's threat model
stated by someone who already shipped it: *"If repos can define their own workflows, then anyone
that can create a pull request to that repo can essentially run arbitrary code on your Atlantis
server."* Their answer was a **deny-by-default allowlist** — `allowed_overrides` names exactly what
a governed repo may override — rather than a denylist of abuses. Same conclusion as §4.2's
allowlist-over-denylist argument, reached independently.

Second, and this is the project's opening. `source` accepts only `local`, and Atlantis's own example
config carries this comment verbatim:

```yaml
      path: /home/atlantis/conftest_policies  # Consider separate vcs & mount into container
```

Remote policy *is* documented, but not via `source` — it delegates to Conftest's puller:

> *"Conftest supports pulling policies from remote locations such as S3, git, OCI, and other
> protocols supported by the go-getter library. The key `extra_args` can be used to pass in the
> `--update` flag."*

```yaml
    policy_check:
      steps:
        - policy_check:
            extra_args: ["--update", "s3::https://s3.amazonaws.com/bucket/foo"]
```

**So the most popular open-source PR-gating policy engine has no first-class concept of a remote
policy source.** Remote policy is a per-workflow `extra_args` string passed to a subprocess, with
authentication left to ambient host credentials, and the alternative suggested in its own docs is
bind-mounting a second Git repository into the container. **Central policy distribution is not
modelled; it is improvised. That is precisely where this project's §3 goes, and it is the clearest
evidence that the gap is real.**

Operational details worth pre-empting: conftest runs only the `main` package by default; *"Conftest
is run with an argument vector rather than through a shell"* so *"Shell operators and command
substitution are passed through as ordinary text"* (someone fixed an injection bug there); and
policy checks are skipped for manual `atlantis plan` unless allowlisted, while autoplans bypass the
check *"because they don't have a user"* — the same actor-dependence problem §6.1's audit
discussion raises.

### 6.4 Terraform plan policy — the input-schema lesson

This is the canonical "OPA over a structured diff" pattern and the closest analogue to evaluating a
PR diff.

**The plan JSON** ([terraform/internals/json-format](https://developer.hashicorp.com/terraform/internals/json-format)).
`terraform show -json <FILE>` emits top-level `format_version`, `prior_state`, `configuration`,
`planned_values`, `variables`, `resource_changes`, `resource_drift`, `output_changes`, `checks` and
more. Each `resource_changes` element carries `address`, `previous_address`, `module_address`,
`mode` (`managed`|`data`), `type`, `name`, `index`, `deposed`, `change`, `action_reason`. The
`change` object carries `actions`, `before`, `after`, `after_unknown`, `before_sensitive`,
`after_sensitive`, `replace_paths`, `importing`.

**`actions` is a closed enum and its array order is meaningful:**
`["no-op"]`, `["create"]`, `["read"]`, `["update"]`, `["delete","create"]`, `["create","delete"]`,
`["delete"]`.

**This is the single most transferable design in this document.** HashiCorp did not expose "a diff" —
they exposed a **per-object change record with a closed verb enum**, plus parallel `*_sensitive` and
`*_unknown` masks so policy can distinguish *this value changed* from *we do not know this value
yet* from *we must not log this value*. The policy author never parses text.

Three lessons for this project:

- **The engine never parses the native format.** `terraform show -json` is a separate,
  vendor-supported conversion step. **Define and version your own PR-diff JSON schema, and treat
  "GitHub API → that schema" as a distinct, tested, bounded translation layer.** Policy authors
  write against your schema, not GitHub's API, so GitHub API changes do not break every tenant's
  policy — and this is where §4.3's input bounding belongs.
- **The `*_sensitive` mask is the answer to §5.3's problem.** A policy that detects committed
  secrets must not log them; carrying an explicit sensitivity mask in the input schema is how you
  make that mechanical rather than a matter of care.
- **"Blast radius" scoring is the idiom for test selection.** OPA's
  [Terraform tutorial](https://www.openpolicyagent.org/docs/terraform) assigns weights per resource
  type and action — `"aws_autoscaling_group": {"delete": 100, "create": 10, "modify": 1}` — sums
  them, and authorizes only if the total is below a threshold **and** no IAM resources changed. A
  cheap numeric heuristic for the gradient plus absolute rules for the never-acceptable. Copy both
  halves. Note it exposes **two named decisions over one input**: `terraform/analysis/authz` and
  `terraform/analysis/score`. "Proceed vs route to human" wants the same separation — a decision
  *and* a magnitude, not one boolean.

```bash
terraform plan --out tfplan.binary
terraform show -json tfplan.binary > tfplan.json
opa exec --decision terraform/analysis/authz --bundle policy/ tfplan.json
```

`opa exec --decision <path> --bundle <dir> <input files>` is a first-class CLI for "evaluate a named
decision from a bundle against input documents" — the project's CLI (§6.1) in one command, working
against a **signed bundle**, and hence also the local dry-run path from §1.6.

**HashiCorp Sentinel** names the three-level model this project needs
([enforcement-levels](https://developer.hashicorp.com/sentinel/docs/concepts/enforcement-levels)):

| Level | Verbatim |
|---|---|
| **Advisory** | *"The policy is allowed to fail. However, a warning should be shown to the user or logged. Advisory is the default enforcement level."* |
| **Soft Mandatory** | *"The policy must pass unless an override is specified."* Overrides *"enable privilege separation and provide non-repudiation through explicit actor acknowledgment."* |
| **Hard Mandatory** | *"The policy must pass no matter what. The only way to override a hard mandatory policy is to explicitly remove the policy."* |

**That soft-mandatory justification — *privilege separation and non-repudiation through explicit
actor acknowledgment* — is this project's "route to human review" outcome, named and justified by
the vendor. Use their words.** (**Unverified**: Sentinel's exact license. No HashiCorp licensing
statement appears on the Sentinel docs; what is verifiable is that source is unpublished and it is
documented only as a feature of HashiCorp products. The flat claim "proprietary, closed-source"
traces to third-party blogs, so §7.5's wording should be read with that caveat.)

**HCP Terraform supports OPA natively**, with **advisory** (*"Failed policies never interrupt the
run"*) and **mandatory** (*"Failed policies stop the run, but any user with Manage Policy Overrides
permission can override these failures"*) levels
([manage-policy-sets](https://developer.hashicorp.com/terraform/cloud-docs/policy-enforcement/manage-policy-sets)).
Policy reaches it by Web UI, **version control** (*"When you push changes to the repository, HCP
Terraform automatically uses the updated policy set"*), or API, and attaches *"globally, to specific
projects, workspaces, and Stacks, or to workspaces with specific tags"*, with exclusions. The VCS
format is a `policies.hcl` at the repo root:

```hcl
policy "policy_name" {
   query = "data.terraform.policy_name.deny"
   enforcement_level = "mandatory"
   description = "policy description"
}
```

Two load-bearing lessons:

- **`input.plan` and `input.run` are siblings.** *"To access the properties from the Terraform plan
  data in your policies, use `input.plan`. To access properties from the Terraform run, use
  `input.run`."* The artifact and the context of who/what/where are **separate top-level keys**. The
  project's equivalent is `input.diff` alongside `input.pr` (author, base, repository, labels,
  whether the author is a bot or an agent). **Do not flatten these** — two independent vendors
  converged on the split (see Spacelift below).
- **The contract is "an array of messages; empty means pass"**, not a boolean: *"The result of each
  query must return an array... If the array is empty, HCP Terraform reports that the policy has
  passed."* Violations carry their own explanations, which is what makes "route to human review"
  actionable rather than a bare red X.

**And the meta-lesson, which resolves a contradiction in HashiCorp's own docs:** their framework
comparison page says OPA is *"Pass/fail only - no built-in concept of advisory or soft-mandatory
levels"*, while manage-policy-sets documents advisory/mandatory for OPA policy sets. The
reconciliation — Rego-the-language has no enforcement-level concept, and HCP layers it on via
`enforcement_level` in `policies.hcl` — is **inference, not quoted**. But the lesson survives either
way and is the key one: **enforcement level is a property of the harness, not of the policy
language.** Which is exactly why §6.1's `scoped`/`scopedEnforcementActions` belongs in the project's
own policy metadata.

### 6.5 Spacelift — Rego gating a plan *and* routing to human review

Eight Rego policy types ([docs.spacelift.io/concepts/policy](https://docs.spacelift.io/concepts/policy/)):
Login, Access, **Approval**, Notification, **Plan**, Push, Trigger, Intent. Engine is OPA; Rego v1
recommended.

**Plan policy has exactly two rules, and the `warn` definition is the one to note:**

- **`deny`** — *"Print in red. Automatically fail the run."*
- **`warn`** — *"At most, mark the run for human review if the change affects the tracked branch and
  the stack is set to autodeploy."*

**Three outcomes from two rule names plus harness context.** The policy author states *severity*;
the *harness* decides whether severity-2 means block or means fetch a human, based on blast radius.
**This project's proceed/review/deny split almost certainly belongs on the same seam** — and it
means policy authors never need to know whether a given deployment is configured to auto-merge.

Input namespacing mirrors HashiCorp's: `input.terraform.resource_changes[]` for the artifact and
`input.spacelift.{commit,run,stack}` for context (`run.type` ∈ `"PROPOSED"`/`"TRACKED"`; `stack`
carries `autodeploy` and `labels`), plus `input.third_party_metadata.custom`.

**Approval policy is a *separate* policy type over a *different* input** — two boolean rules
`approve` and `reject`, with `input.reviews.current.{approvals,rejections}` carrying reviewer
username, roles and resolved identity:

```rego
approve if input.run.state != "UNCONFIRMED"

approve if {
  count(input.reviews.current.approvals) > 1
  count(input.reviews.current.rejections) == 0
}
```

*"When a queued run needs approval, it will not be scheduled until that approval is received."*

**And the attachment mechanism is the cleanest multi-tenant design in the corpus:**
**`autoattach:<label>` special labels** (wildcards allowed, e.g. `autoattach:*`). Onboarding a
repository needs no policy-side change, and **a tenant cannot detach a policy by editing their own
repository**, because policy and attachment both live in the control plane. Compare §3.1's bundle
`roots`: `roots` isolates *what a bundle may define*, `autoattach` decides *which tenants a policy
applies to*. This project needs both, and they are different mechanisms.

**Lessons:** approval *routing* can itself be policy — but keep it a separate policy type with its
own input, not a second rule inside the gating policy; and label-selector attachment beats
per-tenant configuration.

### 6.6 GitHub Actions, and GitHub's own primitives

**GitHub Actions is structurally the wrong enforcement point, and the ecosystem demonstrates it.**
Verified against the GitHub API rather than marketplace listings:

| Action | Status | Verdict |
|---|---|---|
| `open-policy-agent/setup-opa` | Apache-2.0, active (last push 2026-07-21) | ✅ Real, official, maintained |
| `instrumenta/conftest-action` | Last push 2021-11-29 | ⚠️ Abandoned |
| `artis3n/setup-conftest` | **Archived** 2021 | ❌ Dead |
| `koozz/opa-action` | **Archived** Dec 2021, self-deprecating | ❌ Dead |

There is **no official `setup-conftest`**; the OPA org ships `setup-opa` only.

**The most informative finding is a deprecation notice.** `koozz/opa-action` was literally *"OPA
Pull-Request Assessor... a GitHub Action that checks files against policies (configured in the same
repo)"*. It is archived, and its README calls the action obsolete, recommending users *"simply use
Conftest directly"*. Note **"configured in the same repo"** — the one prior attempt at a PR assessor
as an Action put policy **inside the governed repository**, the arrangement this project's thesis
rejects, and it died.

**The structural argument, which is architectural rather than ergonomic: a GitHub Action runs inside
the governed repository's own workflow file, which that repository's developers — and their coding
agents — can edit. An Action can never be the enforcement point for a policy its target is not
allowed to change.** A webhook-driven external service can. The Actions ecosystem converged on
"install a CLI, run it in a step", which leaves policy-outside-the-repo unsolved and re-improvised
by every team.

**GitHub rulesets and CODEOWNERS are the incumbents this project displaces**, and they matter
because they are what users have today.

**Rulesets** ([about-rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets))
— *"a named list of rules that applies to a repository or to multiple repositories in an
organization"*, with up to 75 at repo level and 75 org-wide, and *"You can allow certain users to
bypass the rules"* (roles, teams, or **GitHub Apps**). Available rules include "Require a pull
request before merging", "Require status checks to pass before merging", "Require signed commits",
and — note these — **"Restrict file paths"** (fnmatch), "Restrict file extensions", "Restrict file
size". Three consequences, all actionable:

1. **Org-level rulesets already give GitHub users "policy outside the repo."** The pitch cannot be
   *policy outside the repo* alone; it must be ***expressive* policy outside the repo** — rulesets
   cannot express "a diff that changes an auth check and also adds a dependency needs human review".
2. **"Restrict file paths" is how the project protects itself.** If any configuration must live in
   the governed repository, an org-level ruleset restricting that path is what makes it non-editable
   by the developers and agents it constrains. **Document it as a recommended deployment step.**
3. **"Require status checks to pass before merging" is the integration point**, and a GitHub App can
   be granted bypass — which is how the project becomes authoritative without becoming a permanent
   merge bottleneck for break-glass.

**CODEOWNERS** ([about-code-owners](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners))
lives in `.github/`, the repo root, or `docs/`; *"Each CODEOWNERS file assigns the code owners for a
single branch"*; and critically *"For code owners to receive review requests, the CODEOWNERS file
must be on the base branch of the pull request."*

**This is the crispest concrete articulation of the project's thesis available, entirely from
GitHub's own docs.** Base-branch resolution means a PR cannot grant *itself* new code owners
mid-flight — a real, deliberate protection. But CODEOWNERS is still a file **in the governed
repository**, so the *next* PR inherits whatever the *last merged* PR changed it to. **The guard is
exactly one merge deep.** Closing it requires bolting on a path restriction. This project's
architecture closes it by construction. **Lead with this example.**

### 6.7 OPA's own PR-gating guide — and why the ground is unoccupied

OPA ships an official PR-gating guide, and it is the sharpest competitive read available
([cicd/pr-checks](https://www.openpolicyagent.org/docs/cicd/pr-checks)):

```bash
gh api repos/${{ github.repository }}/pulls/${{ github.event.number }}/files --paginate > changed_files.json
opa eval --input changed_files.json --data policy/pr-check/pr_check.rego --format raw "data.pr_check.changes.$check"
```

```rego
changes["frontend"] if {
    some file in input
    startswith(file.filename, "frontend/")
}
```

And the [CI/CD overview](https://www.openpolicyagent.org/docs/cicd) lists OPA's recommended use
cases as **repository governance (validating commit messages and pull request metadata)**, supply
chain validation, **test selection (determining which tests to run based on changed files)**,
coverage enforcement, and job dependencies.

**Read this carefully, because it cuts both ways.** OPA's own canonical PR-gating example takes
**only the list of changed filenames** as input, and its only decision is **which CI jobs to run** —
routing, not admission. The state of the art in OPA-land for pull requests is
`startswith(file.filename, "frontend/")`. **Nobody in the OPA ecosystem has published a structured
PR-diff input document.** So "evaluate Rego against the *structure* of a PR diff" is genuinely
unoccupied ground — **which also means the project owns the input schema and no tenant has prior
expectations to violate.** Spend the design effort there, modelled on how HashiCorp curated
`resource_changes` (§6.4).

Note also that "test selection" is literally in OPA's own list, and is the example this project
chose for fail-open. Their framing supports it: **routing decisions and admission decisions are
different use cases of the same engine and should not share a failure mode.**

The relevant `opa eval` exit-code flags: `--fail` (*"Exits with non-zero exit code on
undefined/empty result and errors"*) and `--fail-defined` (*"Exits with non-zero exit code on
defined/non-empty result and errors"*), plus `--fail-non-empty` on `opa exec`. **Which flag you use
is determined by whether the convention is allow-shaped or deny-shaped**: `allow` → `--fail`;
`deny` → `--fail-defined`; a `deny` set → `--fail-non-empty`. Pick one convention for tenants and
document it, because the two cannot be mixed behind one exit code.

### 6.8 OPA Control Plane — this project's distribution thesis, productised

This is the most important find in this section, and it was not on the original candidate list.

**Context first, which also corrects a likely premise.** On **2025-08-20** OPA's creators posted
*"Note from Teemu, Tim, and Torin to the Open Policy Agent community"*: *"the creators of Open
Policy Agent (along with many team members from Styra) have joined **Apple**..."* and, verbatim,
*"Open Policy Agent remains a CNCF graduated open source project and there are no changes to the
project governance or licensing."* Maintainers unchanged *"except for an organization change from
Styra to Apple"* — which is exactly what `MAINTAINERS.md` shows (§8). Styra's commercial tooling is
being contributed to the CNCF OPA org: **EOPA**, **OPA Control Plane**, several SDKs, and **Regal**
(a Rego linter worth knowing about for policy-author tooling).

(**Unverified / do not record as fact**: any claim that Styra "went out of business" on a particular
date — that traces to a competitor's migration-pitch blog. What is verifiable is the Apple move and
that `styra.com`/`docs.styra.com` no longer resolve as of 2026-10-13 while `openpolicyagent.org`
resolves fine. Styra DAS's internal architecture is therefore secondary and possibly stale, and I
found **no evidence** Styra DAS ever had a CI/PR-gating feature.)

**The live successor is [`open-policy-agent/opa-control-plane`](https://github.com/open-policy-agent/opa-control-plane)
(OCP)**, Apache-2.0, `opactl` CLI, **v0.8.0 published 2026-10-06** (from the Go module proxy, which
is authoritative here). README features, verbatim:

- *"**Git-based Policy Management.** Build bundles based on Rego from multiple Git repositories and
  implement environment promotion strategies natively with Git."*
- *"**External Datasources.** Fetch and bundle external data required by your policies
  build-time."*
- *"**Highly-Available & Scalable Bundle Serving.** Distribute bundles to cloud object storage like
  AWS S3, Google Cloud Storage, or Azure Blob Storage."*
- *"**Global and hierarchical policies.** Enforce organization-wide rules by defining global
  policies that get injected into bundles at build-time **based on label selectors**. Global
  policies can override other policies based on custom conflict resolution logic written in Rego."*

Config shape (`config.d/*.yaml`, built with `opactl build`):

```yaml
bundles:
  hello-world:
    object_storage:
      filesystem:
        path: bundles/hello-world/bundle.tar.gz
    requirements:
    - source: hello-world
sources:
  hello-world:
    directory: files/sources/hello-world
    paths:
    - rules/rules.rego
```

From [ocp/concepts](https://www.openpolicyagent.org/docs/ocp/concepts): **bundles** carry `labels`
(environment, team, system-type); **git sources** declare `repo`, `reference`, `commit`, `path`,
`include_files`, `excluded_files`, `credentials`; **requirements** support `path` and `prefix` for
namespace management; **secrets** support `aws_auth`, `basic_auth`, `gcp_auth`, `azure_auth`,
`github_app_auth`, `ssh_key`, `token_auth`, `password`. And **stacks**: *"Stacks enforce consistent
policy distribution to OPA instances managed by OCP. They automatically inject required sources into
matching bundles at build time."* A stack applies when its selector matches the bundle's labels
**and** its exclude-selector does not, with glob support.

**That is this project's central thesis, implemented, by the people who invented Rego.** A central
policy author defines a stack; it is injected into every matching tenant bundle **at build time**;
the tenant cannot opt out, because the tenant does not run the build. Multi-tenant binding is a
label selector (the same answer Spacelift reached with `autoattach:`). And they named the hard part
explicitly rather than treating it as an edge case: **what happens when a global rule and a tenant
rule disagree** — their answer is custom conflict resolution written in Rego. **§3.6 should expect
to need that, and it is a question the project will otherwise discover late.**

Two caveats: OCP is **early** (0.x, created 2025-04-28, ~67 stars, no production-readiness
statement) and it requires a SQL database (`postgres`/`mysql`/`sqlite3`/`aws_rds`) when run as a
service. **Do not depend on it. Do read it** — it is the closest thing to a reference architecture
for the project's policy-distribution layer, and if it matures, "bring your own OCP" is a plausible
deployment story that removes §3.6 from the project's scope entirely.

### 6.9 Other systems, reported honestly

- **Kyverno** — Apache-2.0, ~8.2k stars, the main non-OPA Kubernetes policy engine. Moved from
  YAML+JMESPath to **CEL**; new `policies.kyverno.io` types (`ValidatingPolicy`, `MutatingPolicy`,
  `GeneratingPolicy`, `ImageValidatingPolicy`, `DeletingPolicy`) with legacy
  `ClusterPolicy`/`Policy` deprecated for removal in v1.20. Enforcement is two-valued: **Audit** and
  **Enforce**. ⚠️ **`kyverno/kyverno-json` was archived 2026-09-21** with the notice *"Kyverno has
  transitioned to CEL... and JMESPath based JSON processing is deprecated."* It had been the
  component applying Kyverno policies to arbitrary JSON/YAML via CLI, REST service, **or Go
  library** — the nearest non-Rego analogue to this project. **Two lessons:** the only serious
  attempt at admission-style policy over arbitrary JSON outside Kubernetes was archived weeks ago
  and folded back into a Kubernetes-shaped CEL product, so the niche is unoccupied again; and
  Kyverno **owns its policy language** and consequently had to migrate its entire user base from
  JMESPath to CEL. **Embedding OPA/Rego means inheriting a stable, CNCF-governed language you do not
  have to steer — put that in the "why Rego" section of the design doc.**
- **Snyk IaC custom rules** — Rego, with a Go SDK (`snyk/snyk-iac-rules`). Workflow
  `template → parse → test → build → snyk iac test --rules=bundle.tar.gz`, and **distribution is OCI
  via ORAS**: `snyk-iac-rules push -r docker.io/example/test:v0.0.1 bundle.tar.gz`, supporting GCR,
  DockerHub, ECR, ACR, Artifactory, Harbor, GHCR, HTTPS only. **OCI-via-ORAS with OPA bundles as
  payload is the convergent answer for distributing Rego** — Conftest, gator, Snyk and OCP all land
  there, which is strong corroboration for §3.5. Note too that every one of these ships a *local*
  test loop (`snyk-iac-rules test`, `conftest test`, `opa test`, `gator verify`); tenant policy
  authors will demand one.
- **Datadog IaC Security custom rules** — *"Datadog IaC custom rules use Rego."* Input is
  `input.document[i].resource`; rules are authored **in the Datadog control plane**, bound to
  scanners by platform, and gated by an explicit `appsec_vm_write` permission. Note the input is a
  **parsed-source** shape, not a plan/diff shape — it answers "is this config bad", not "is this
  *change* bad", which is the weaker form; §6.4's lineage is the stronger one. The transferable bit
  is the permission: **separating "may author policy" from "may author workload" as an explicit,
  named permission** is the mechanism implementing the project's load-bearing separation.
  (**Unverified**: whether Datadog IaC results gate PRs.)
- **Argo CD has no native OPA/Rego support.** Its RBAC is **Casbin**. OPA/Gatekeeper runs
  *alongside* it as a cluster admission controller. Any claim of "Argo CD OPA policy" is wrong.
- **Unverified, do not cite:** Terrateam's OPA/conftest docs now redirect to `stategraph.com`, which
  mentions neither Terrateam nor OPA; the relationship is unknown. Semgrep (own YAML pattern DSL),
  Checkov (Python/YAML checks), Permit.io and Cerbos (application-level authz, not CI gates) and
  Backstage (runtime UI authz via community OPA plugins) were not researched to primary sources.

### 6.10 What to copy, in priority order

1. **The ConstraintTemplate/Constraint split with a schema on the parameters, and binding failures
   that are loud.** It separates who writes the logic from who configures it, is what makes "end
   users author policies" safe, and shrinks the §4 untrusted-code surface. Gatekeeper's own
   regression — a non-structural schema turning a constraint into a silent no-op — is why rejection
   must happen at submit time.
2. **Severity as a per-policy, per-enforcement-point declaration** — Gatekeeper's `scoped` +
   `scopedEnforcementActions`, which is literally the fail-closed-on-security /
   fail-open-on-test-selection requirement already designed. And keep it strictly separate from the
   fail-open/fail-closed axis.
3. **The three-way failure split**: cannot reach the engine → allow (availability); engine ran and
   errored → deny (correctness); self-exemption attempt → always deny (integrity). This replaces
   §4.4's two-way split.
4. **Exempt your own repositories and build a break-glass path outside the enforced system, now** —
   and make sure your deployment automation will not immediately undo it.
5. **Audit mode**: periodic re-evaluation of existing state, capped detail with an uncapped count,
   as a singleton. It is the rollout-safety number, the drift detector, the dashboard, and the thing
   that makes failing open defensible.
6. **A versioned diff-JSON schema as the policy input contract** — per-object change records, a
   closed action enum, and explicit `sensitive`/`unknown` masks; with `input.diff` and `input.pr` as
   siblings, and GitHub-to-schema as a separate bounded layer.
7. **Compile-time containment of tenant Rego**: a capability allowlist (§4.2), `Externs` restricted
   to the one data path you intend, and package-path rewriting or bundle `roots` for namespace
   isolation.
8. **A `gator`-style CLI pulling the same signed artifact the server enforces**, serving both
   policy-author tests (with a denial-test primitive) and developer pre-flight checks — plus a
   `gator bench`-style latency-regression gate in the policy repository's CI.
9. **Label-selector attachment** (Spacelift's `autoattach:`, OCP's stack selectors) so onboarding a
   repository needs no policy-side change and a tenant cannot detach a policy.
10. **Approval bound to the evaluated artifact, not the PR** (Atlantis), with `prevent_self_approve`
    and per-policy owners.
11. **Do not import `frameworks/constraint` or Conftest** — but read `tftarget` and
    `gcv/validator.go` first. Conftest additionally has a disqualifying `opa.runtime().env`
    exposure, disk-only policy loading, a racy `Check`, no prepared queries, and 165 dependencies.
12. **Read OPA Control Plane before designing §3** — especially its answer to global-versus-tenant
    policy conflict, which the project will otherwise discover late.

---

## 7. Alternatives to OPA, Assessed Honestly

The brief asks for a defensible choice, not a justification. So: **OPA wins this on exactly one
axis — policy lifecycle — and loses on another that matters a great deal here, namely resource
bounding of untrusted policy.** The honest summary table first, then the reasoning.

| | Native Go | Termination guaranteed | CPU bound | Memory bound | Collection reasoning | Aggregation (sum/count) | Distribution + signing | License |
|---|---|---|---|---|---|---|---|---|
| **OPA / Rego** | yes | yes (recursion is a compile error) | **no** | no | full | full | **bundles + JWS signing, built in** | Apache-2.0 |
| **CEL** | yes | **yes, by language design** | **yes (`CostLimit`, `EstimateCost`)** | partial (`OptTrackMemory`) | `all`/`exists`/`filter`/`map` macros | **no `sum`/`reduce`** | **none — you build it** | Apache-2.0 |
| **Cedar (cedar-go)** | yes (native port) | yes (no loops exist) | n/a | n/a | **none** | **none, not even `count`** | none | Apache-2.0 |
| **Starlark** | yes | config-dependent (safe by default) | yes (step limit) | **no — explicitly none** | full (it is Python) | full | none | BSD-3-Clause |
| **Go predicates** | n/a | yes | n/a | n/a | full | full | **n/a — needs redeploy** | yours |

### 7.1 CEL (Common Expression Language)

**Module path — this has moved, and getting it wrong is easy.** Historically
`github.com/google/cel-go` (v0.2.0–v0.31.0); the canonical path for v0.32.0 and later is
**`cel.dev/cel-go`** (v0.32.0, published 2026-08-19). The repository moved from
`github.com/google/cel-go` to [`github.com/cel-expr/cel-go`](https://github.com/cel-expr/cel-go),
which carries the banner *"On June 16, 2026, this repository will move to
github.com/cel-expr/cel-go!"*. Pure Go. (**Unverified**: cel-go makes no explicit "no CGO"
statement the way wazero does; its dependency set — ANTLR Go runtime, `google.golang.org/protobuf`,
`golang.org/x/text` — is pure Go, so a CGO-free build is a high-confidence inference, not a quoted
guarantee.)

The API, verified on pkg.go.dev, has the same compile-once/evaluate-many shape as OPA:

```
cel.NewEnv(opts ...EnvOption) (*Env, error)
cel.NewCustomEnv(opts ...EnvOption) (*Env, error)   // no standard library
(*Env).Compile(txt string) (*Ast, *Issues)          // parse + check
(*Env).Program(ast *Ast, opts ...ProgramOption) (Program, error)
(Program).Eval(activation Activation) (ref.Val, *EvalDetails, error)
```

One `Program` is reusable across goroutines, exactly like a `PreparedEvalQuery`.

**CEL's decisive advantage is resource bounding, and it is a real one.** The CEL language
definition states flatly that CEL is *"terminating: CEL programs cannot loop forever"* —
`for`/`while`/`loop`/`break`/`continue` are reserved-but-unimplemented, and there is *"no way to
write recursive functions"* because user-defined functions do not exist. Comprehension macros
iterate finite collections only. The spec does warn that nested macros can produce *"exponential
behavior"*, bounded by input size but not cheap
([cel-spec langdef](https://github.com/google/cel-spec/blob/master/doc/langdef.md)).

On top of the language guarantee, cel-go provides a runtime budget. From
[`cel/options.go`](https://raw.githubusercontent.com/cel-expr/cel-go/master/cel/options.go):

```go
// CostLimit enables cost tracking and sets configures program evaluation to exit early with a
// "runtime cost limit exceeded" error if the runtime cost exceeds the costLimit.
// The CostLimit is a metric that corresponds to the number and estimated expense of operations
// performed while evaluating an expression. It is indicative of CPU usage, not memory usage.
func CostLimit(costLimit uint64) ProgramOption

func CostTracking(costEstimator cost.ActualCostEstimator) ProgramOption
func CostTrackerOptions(costOpts ...cost.TrackerOption) ProgramOption

// InterruptCheckFrequency configures the number of iterations within a
// comprehension to evaluate before checking whether the function evaluation
// has been interrupted.
func InterruptCheckFrequency(checkFrequency uint) ProgramOption
```

plus static pre-deployment estimation:

```
(*Env).EstimateCost(ast *Ast, estimator checker.CostEstimator, opts ...checker.CostOption) (checker.CostEstimate, error)
```

`EvalOption` constants include `OptTrackCost` and `OptTrackMemory`; `EvalDetails.ActualCost()
*uint64` returns measured cost. (Version skew to watch: older pkg.go.dev pages show
`CostTracking(interpreter.ActualCostEstimator)`; current source uses `cost.ActualCostEstimator` —
the estimator interface changed packages.)

**Compare §4.3.** CEL offers a hard per-evaluation CPU budget *and* a static worst-case estimate
you can use to reject a policy before ever running it. OPA offers neither — only a `context`
deadline, which stops the work only after it has already been done. For a project whose engine
runs tenant-authored code, this is the single strongest argument against OPA on this list.

**Kubernetes ValidatingAdmissionPolicy is the closest prior art to this project's architecture,**
and its calibration is a gift. From
[validating-admission-policy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/):
`spec.matchConstraints.resourceRules` for selection; `spec.validations[].expression` (must be
`true` to pass) with `reason`, `message`, `messageExpression`; `spec.failurePolicy` (`Fail`
default, or `Ignore`); `spec.matchConditions` as cheap CEL pre-filters; and two features worth
copying outright:

- **`spec.paramKind`** — a parameter resource (ConfigMap or CRD) referenced in expressions as
  `params`, e.g. `object.spec.replicas <= params.maxReplicas`. This is "one central policy
  definition, tenants supply the knobs" — the same idea as Gatekeeper's ConstraintTemplate/Constraint
  split (§6) and the right answer to this project's multi-tenancy.
- **`spec.variables`** — named expressions referenced as `variables.foo`, **lazily evaluated on
  first reference, with results *and errors* cached** so each counts once against the cost budget.
  Declaration order is significant, which is what structurally prevents circular references.

And the policy/binding split: `ValidatingAdmissionPolicyBinding.spec.validationActions` ∈
`Deny` / `Warn` / `Audit` (`Deny` and `Warn` mutually exclusive). One policy, many per-scope
bindings — directly applicable as "one policy, many per-repository bindings".

The cost constants Kubernetes imposes, read from
[`staging/src/k8s.io/apiserver/pkg/apis/cel/config.go`](https://raw.githubusercontent.com/kubernetes/kubernetes/master/staging/src/k8s.io/apiserver/pkg/apis/cel/config.go):

| Constant | Value | Documented meaning |
|---|---|---|
| `PerCallLimit` | `1000000` | Cost limit per CEL validation call; "roughly 0.1 second" |
| `RuntimeCELCostBudget` | `10000000` | Overall budget per binding or CustomResource; "roughly 1 second" |
| `RuntimeCELCostBudgetMatchConditions` | `2500000` | Budget for `matchConditions` per object; "roughly 0.25 second" |
| `CheckFrequency` | `100` | Comprehension iterations between interrupt checks |
| `MaxRequestSizeBytes` | `3145728` (3 MB) | Max API server request size |
| `MaxEvaluatedMessageExpressionSizeBytes` | `5120` (5 KB) | Cap on generated `messageExpression` output |

CEL costs are **deterministic** — the same expression and input yield the same cost regardless of
load. A `<` comparison is 1 unit; a list literal 40+; regex is approximated as
`length(regexString) * length(inputString)`. Kubernetes enforces both a runtime budget (halt with
an error) and estimated cost limits (reject the expression at admission time if the worst case
would be prohibitive) ([Kubernetes CEL docs](https://kubernetes.io/docs/reference/using-api/cel/)).
**`PerCallLimit = 1000000 ≈ 0.1s` is an empirically tuned cost↔wallclock mapping from a project
running this at enormous scale. Adopt those numbers rather than guessing — that is worth reading
even if you stay on OPA, because it tells you what a sane per-decision budget looks like.**

**Expressiveness for a diff.** The standard macros `has`, `all`, `exists`, `exists_one`, `map`,
`filter` are available everywhere, so the canonical shapes work:

```cel
// "more than 5 files under /infra changed"
object.files.filter(f, f.path.startsWith("infra/")).size() > 5

// "no file outside the allowlisted directories"
object.files.all(f, allowedPrefixes.exists(p, f.path.startsWith(p)))

// "any deletion of a protected workflow file"
object.files.exists(f, f.changeType == "DELETE" && f.path.matches("^\\.github/workflows/"))
```

Extension libraries ([pkg.go.dev/.../cel-go/ext](https://pkg.go.dev/github.com/google/cel-go/ext))
add a lot: `ext.Lists()` (`distinct`, `flatten`, `sort`, `sortBy`, `slice`, `reverse`, `first`,
`last`), `ext.Sets()` (`sets.contains`, `sets.equivalent`, `sets.intersects`), `ext.Strings()`,
`ext.Regex()`, `ext.Math()`, `ext.Bindings()` (the `cel.bind()` macro), `ext.Encoders()`, and
`ext.TwoVarComprehensions()`.

**The real gap: there is no `sum`, `reduce` or fold in base CEL or any extension library.**
`math.greatest`/`math.least` give you min/max over a list, but "total added lines across files
matching X" is **not expressible**. Two honest workarounds: register a host function from Go, or
**precompute the aggregate in Go and put it in the input document** (`input.diff.totalAdditionsUnderInfra`).
The second is better for this project regardless of engine — it keeps cost predictable and the
policy readable. But if tenants need to author *arbitrary* aggregations you have not anticipated,
that is a genuine point for Rego, which has `sum`, `count`, `max`, aggregating comprehensions and
partial set/object rules natively.

**Distribution — CEL's weakest card, and the reason OPA wins.** Bare CEL has no module system, no
bundle format, no signing, no revision metadata, no distribution protocol. None of it.

The gap is narrower than most write-ups admit, though, because of
[`github.com/google/cel-go/policy`](https://pkg.go.dev/github.com/google/cel-go/policy) — *"an
extensible parser and compiler for composing a graph of CEL expressions into a single evaluable
expression"*, with a YAML authoring format: policy-level `name`, `description`, `imports`, `rule`;
`rule` holding `variables` (lazily evaluated, memoized, declared before use) and `match` blocks;
`match` holding `condition`, `output`, or a nested `rule`, with top-down first-match semantics.
Exported: `Compile()`, `CompileRule()`, `NewParser()`, `FromConfig()`, and types `Policy`, `Rule`,
`Match`, `Variable`, `CompiledRule`, `RuleComposer`. **It even ships a test-suite format**
(`TestSuite`, `TestSection`, `TestCase`, `TestInput`). So "no composition" and "no test framework"
are no longer fair criticisms. (**Unverified**: this is a v0.x subpackage with no stability
statement found; assume the API can move.)

What genuinely remains for you to build: an artifact format, a revision/manifest concept,
**signing and verification**, a fetch-activate-rollback loop, and multi-tenant scoping. That is
§3 of this document, in its entirety, written by you. **Budget it honestly — it is weeks of
security-critical supply-chain work, and it is the entire reason OPA wins.**

Tooling is good but fragmented: an official `cel-go/repl`, but the best IDE pieces
(the [`cells`](https://github.com/stefanvanburen/cells) language server,
[CEL Playground](https://github.com/undistro/cel-playground), `vscode-cel`) are
one-maintainer third-party projects. Apache-2.0. Adoption risk is low — Kubernetes, Envoy and
Kyverno all depend on it — though (**unverified**) I found no formal governance charter or CNCF
status; the `google/` → `cel-expr/` org move suggests a deliberate step toward vendor neutrality.

**Verdict.** Gains: a hard CPU budget plus static worst-case estimation; a much smaller attack
surface (no `http.send`-equivalent to strip); Kubernetes-validated cost constants and an exemplary
policy/binding/params architecture; a far smaller dependency footprint. Costs: you build the
entire policy supply chain; no tenant-authorable aggregation; weaker IDE story; less familiar to
an audience that already writes Rego.

### 7.2 AWS Cedar

Cedar is an open-source authorization language from AWS built around `permit`/`forbid` statements
scoped by **principal, action, resource** with a `when`/`unless` clause over a `context` record.
The core is **Rust** ([cedar-policy/cedar](https://github.com/cedar-policy/cedar), Apache-2.0),
and its headline feature is automated reasoning:
[cedar-spec](https://github.com/cedar-policy/cedar-spec) contains a Lean formalization and proofs,
validated against the Rust implementation by differential randomized testing, and
`cedar-policy-symcc` is a symbolic compiler.

**The Go story is better than expected.**
[`github.com/cedar-policy/cedar-go`](https://github.com/cedar-policy/cedar-go) exists, sits in the
official `cedar-policy` org, and is a **native Go reimplementation — not CGO, not WASM.** It
originated at StrongDM and was adopted into the official org; latest release observed v1.8.0
(2026-06-01), Apache-2.0. Backward compatibility is maintained for Go minor versions released
within 6 months of a cedar-go release; `x/exp` is exempt from semver.

Included per the README: *"the core authorizer, JSON marshalling and unmarshalling, all core and
extended types (including RFC 80's datetime and duration), integration test suite, schema parsing
and programmatic construction."*

**Not included:** *"CLI applications, the schema validator (experimental support is provided in
x/exp/schema), the formatter, partial evaluation, support for policy templates."*

Read that exclusion list in the context of tenant-authored policy. **The validator — the thing
that type-checks a tenant's policy against your schema before you ever run it — is
experimental-only in Go. Policy templates, the mechanism you would use to parameterize one
central policy across many tenants, are absent entirely.** Those are precisely the two features a
multi-tenant policy product needs most. Further, cedar-spec's Lean proofs target the *Rust*
implementation; the Go port is not mentioned. **The formal-verification selling point does not
transfer** — you would be running an unverified reimplementation of a verified specification.
(**Unverified**: the cedar-go README does not state which Cedar *language* version it implements,
and I found no published mapping of cedar-go releases to Rust `cedar-policy` versions. For a
project where policy is authored centrally and must behave identically across tooling, resolve
that before committing.)

**The expressiveness mismatch is fatal, and it is not a close call.** From the official
[operator reference](https://docs.cedarpolicy.com/policies/syntax-operators.html), Cedar's set
operators are `.contains()`, `.containsAll()`, `.containsAny()`, `.isEmpty()`, plus `in`
(hierarchy membership), `has` (attribute presence), and `.hasTag()`/`.getTag()`.

**There is no iteration, no quantification, no comprehension, no filter, no map, no aggregation —
and no set size or count operator at all.**

So this project's own example — "block this PR if more than 5 files under `/infra` changed" —
**cannot be written in Cedar.** Not awkwardly; not at all. The only route is to compute the answer
in Go and pass it as context:

```cedar
forbid (principal, action == Action::"merge", resource)
when { context.infraFileCount > 5 };
```

At which point your Go code is doing the policy reasoning and Cedar is doing a threshold
comparison. **Every new question a tenant wants to ask requires a new host-computed context field
and a binary redeploy** — which destroys the one requirement that motivated having a policy engine
at all. The omission is deliberate: it is what keeps Cedar decidable for SMT analysis.
(**Unverified**: the operators page does not state that rationale; I infer it from the existence of
`cedar-policy-symcc` and the docs' emphasis on decidability.)

Everything else, briefly. API from the README quickstart: `cedar.Policy` with `UnmarshalCedar()`,
`cedar.NewPolicySet()` / `ps.Add()`, `cedar.EntityMap`,
`cedar.Request{Principal, Action, Resource, Context}`, `cedar.NewEntityUID()`,
`cedar.Authorize(ps, entities, req)`. Termination is trivially guaranteed — there is no looping
construct. No bundle format, no signing, no artifact protocol. `cedar-language-server` and the
formatter are Rust-only, so Go users get neither. (**Unverified**: no published cedar-go benchmark
figures; sub-millisecond is a safe expectation but not a cited one.)

**Verdict: rule Cedar out.** It is a good language for the wrong question — "may this principal
take this action on this resource", not "does this document satisfy these structural
constraints". The entity/action model fights the problem rather than fitting it, and the features
that would have mitigated that (validator, templates) are the ones missing from Go.

### 7.3 Starlark (`go.starlark.net/starlark`)

A deterministic, hermetic Python dialect from Bazel, also used by Buck and Tilt. Pure Go,
**BSD-3-Clause** ([google/starlark-go](https://github.com/google/starlark-go)). The
[spec](https://raw.githubusercontent.com/bazelbuild/starlark/master/spec.md) advertises exactly
the properties you would want: *"The language is deterministic and hermetic. Executing the same
file with the same interpreter leads to the same result. By default, user code cannot interact with
the environment."* And on concurrency: *"Starlark is suitable for use in highly parallel
applications. An application may invoke the Starlark interpreter concurrently from many threads,
without the possibility of a data race, because shared data structures become immutable due to
freezing."*

The spec even claims termination: recursion is a dynamic error, *"all loops are iterations over
finite sequences"*, and *"Execution is finite. The language does not allow recursion or unbounded
loops"* — placing it nominally in the same category as CEL and Rego. **Two things spoil that in
practice.**

**First, starlark-go makes the restrictions optional, per file.** From
[`syntax.FileOptions`](https://pkg.go.dev/go.starlark.net/syntax#FileOptions):

```go
type FileOptions struct {
    Set               bool // allow references to the 'set' built-in function
    While             bool // allow 'while' statements
    TopLevelControl   bool // allow if/for/while statements at top-level
    GlobalReassign    bool // allow reassignment to top-level names
    LoadBindsGlobally bool // load creates global not file-local bindings (deprecated)
    Recursion         bool // disable recursion check for functions in this file
}
```

The zero value is the safe dialect, so `While` and `Recursion` are off by default. But termination
is a **configuration property** here, not a language property — one misconfigured struct literal
from `while True: pass`. CEL cannot be configured into Turing-completeness; Starlark can.

**Second, and decisively, the maintainers tell you not to do this.** From the starlark-go README:

> *"It is trivial for a Starlark program to consume all available time and memory."*
>
> *"if you evaluate Starlark code from an untrusted source in the same address space as your
> application, you are trusting the health of your application to that source."*
>
> **"There is no way to bound the memory used by a Starlark thread."**
>
> *"If you need to defend against denial of service, the only reliable approach is to let the
> operating system help you: evaluate Starlark in a separate process with a tight limit on memory
> (e.g. using ulimit, setrlimit, or cgroups), terminate it if it takes too much wall or CPU time,
> and handle OOM failures, timeouts, and crashes in the parent."*

Single over-large allocations are rejected, but *"a thread can allocate an unbounded amount of
memory in smaller steps."* **"Single self-hostable Go binary, no sidecar" and "the only reliable
approach is a separate process with cgroups" are directly contradictory requirements.**

The step limit does bound CPU, from `starlark/eval.go`:

```go
// SetMaxExecutionSteps sets a limit on the number of Starlark computation
// steps that may be executed by this thread. If the thread's step counter
// exceeds this limit, the interpreter calls the optional OnMaxSteps function
// or the default behavior of calling thread.Cancel("too many steps").
func (thread *Thread) SetMaxExecutionSteps(max uint64)
func (thread *Thread) ExecutionSteps() uint64
```

with `Thread.Cancel(reason string)` safe to call from any goroutine. The `Steps` field's own
comment is candid: *"The precise meaning of 'step' is not specified and may change"* — so a limit
tuned today may mean something different after an upgrade. **It bounds nothing about memory.**

On expressiveness Starlark is the most capable on this list — it is Python, so iteration,
comprehensions, helper functions and `sum()` are all trivial. That is also the cost: an imperative
script has no analyzable structure. You cannot statically determine which fields a policy reads,
cannot estimate cost, cannot explain *why* it denied unless the author built that in, cannot diff
two policies' meanings, and cannot partially evaluate. Rego's `deny` set-of-reasons idiom gives
multi-reason explanations for free; in Starlark every tenant invents a different convention.

Distribution: `load()` exists but **resolution is entirely the host's job**. No bundle format,
manifest, signing, or revision metadata. `go.starlark.net/starlarktest` provides a thin `assert`
module wired to `testing.T`. The project reserves *"the right to make breaking language and API
changes at this stage."*

**Verdict: rule out for untrusted multi-tenant policy** — the absent memory bound is
disqualifying against the single-binary constraint. Keep it in mind only if policy authors become
*trusted* (your own team, or a single-tenant deployment where the operator writes the policy),
where its expressiveness is a genuine asset.

### 7.4 Policy predicates written directly in Go

**The upside is real and nothing competes with it.** Zero marshalling, zero interpretation,
nanosecond decisions. The compiler type-checks your diff model, so a policy referencing a
nonexistent field does not build. Testing is ordinary table-driven Go with coverage, race
detection and fuzzing. **There is no untrusted-code problem because there is no untrusted code** —
no sandbox, no cost budget, no termination question. Debuggers and profilers work normally. You
delete an entire risk category and an entire dependency tree.

For a single-tenant tool whose policy is authored by the team shipping the binary, this is simply
the correct answer, and it is not close.

**Why it collapses here:** tenants author policy, policy is distributed from outside the governed
repo, and it must update without redeploying. Compiled predicates fail all three. A policy change
becomes PR → review → build → release → every self-hoster upgrades.

**Go plugins are not the escape hatch.** Verified from the official
[`plugin` package docs](https://pkg.go.dev/plugin):

- *"Plugins are currently supported only on Linux, FreeBSD, and macOS, making them unsuitable for
  applications intended to be portable."*
- *"Runtime crashes are likely to occur unless all parts of the program (the application and all
  its plugins) are compiled using exactly the same version of the toolchain, the same build tags,
  and the same values of certain flags and environment variables."*
- *"Similar crashing problems are likely to arise unless all common dependencies of the
  application and its plugins are built from exactly the same source code."*
- *"A plugin is only initialized once, and cannot be closed."*
- Plus documented poor race-detector support and **security risks from loading untrusted
  libraries**. The warnings section concludes by suggesting conventional IPC instead.

Every line is fatal here. No Windows. A plugin built with Go 1.27.1 crashes a binary built with
1.27.2, so every tenant artifact is pinned to one exact patch toolchain *and* identical source for
every shared dependency. Policy cannot be hot-reloaded or rolled back, because plugins never
unload. And loading a tenant's `.so` is loading **arbitrary native code with full process
privileges** — no sandbox, no cost limit, no termination guarantee, direct access to the
filesystem, the network, and every other tenant's memory. It is worse than the problem it solves.
**Non-starter, unambiguously.**

**The "config not code" middle path** deserves more respect than it usually gets. A constrained
YAML/JSON rule schema interpreted by Go:

```yaml
rules:
  - name: infra-changes-need-review
    when:
      pathPrefix: "infra/"
      changedFileCountAtLeast: 6
    then: require_human_review
```

What it gains: everything §4 was about, for free. Termination is guaranteed by construction —
there is no interpreter, just a bounded walk over a bounded document. No sandbox needed.
Validation is ordinary boundary validation. Distribution is trivially solved: it is a file, so
sign it with cosign or a detached JWS and verify before activation. Decisions stay in nanoseconds.
Errors are reported at parse time with exact field paths. **Note that this satisfies the
"update without redeploying" requirement that compiled predicates fail** — it is config, so you
hot-reload it.

When it collapses: at the third feature request needing composition. First `anyOf`/`allOf`. Then
negation. Then comparing two fields, so you add an expression micro-syntax. Then cross-rule
references, so you add variables. **At that point you have written a worse CEL, and you own the
parser, type-checker, error messages, formatter, test runner and docs** — all of which CEL or OPA
would have given you.

The non-obvious good design is the hybrid, which is exactly what Kubernetes `paramKind` does: **a
declarative schema for the 90% of policy that is thresholds and path patterns, plus a single
escape-hatch field holding an expression for the rest.** Safety and clarity of config for the
common case; a cost-limited engine only on the path that needs one.

### 7.5 Brief mentions

**CUE (`cuelang.org/go`)** — Pure Go. `cue.Context` (*"used for creating CUE Value objects"*),
`cue.Value` (*"immutable and all methods can be invoked concurrently"*), `Context.CompileString()`,
`Value.Unify()`, `Value.Validate()`, `Value.Eval()`, `Value.Fields()`, plus a `DisallowCycles()`
option ([pkg.go.dev/cuelang.org/go/cue](https://pkg.go.dev/cuelang.org/go/cue)). **Alone among
the alternatives it has a real distribution story**: modules rooted at `cue.mod/module.cue` via
`cue mod init`, a Central Registry at `registry.cue.works`, OCI-registry-backed publishing, and
`CUE_REGISTRY` configuration
([modules docs](https://cuelang.org/docs/concept/modules-packages-instances/)). Its model is
constraint unification — outstanding for "does this document conform to this schema", awkward for
"count the things matching a pattern and compare to a threshold", and it offers no reason-strings
for a decision. (**Unverified**: no documented evaluation cost limit, step limit, or termination
guarantee for CUE — a serious gap for untrusted input, since unification is computationally rich.
License also unverified this pass.) **Verdict: best distribution story on the list, wrong
computational model, no documented resource bound.**

**JSONLogic** — No official Go implementation. The most-used third party is
[`github.com/diegoholiveira/jsonlogic/v3`](https://github.com/diegoholiveira/jsonlogic), MIT, pure
Go, `jsonlogic.Apply(logic, data, &result)` with `AddOperator()` for custom ops. Roughly 225
stars, single maintainer, active CI. **No documented safety constraints, resource limits, or
concurrency guarantees.** No type checker, no schema, no reason mechanism, no module system, no
bundle or signing story, no test framework. **Verdict: strictly less capable than CEL with no
resource bounds. No reason to pick it.**

**Kyverno / `kyverno-json`** — **`kyverno-json` is dead: archived 2026-09-21, read-only.** Its own
README warned *"Kyverno has transitioned to CEL (Common Expressions Language) and JMESPath based
JSON processing is deprecated."* Kyverno proper moved to CEL in 1.14 with new `ValidatingPolicy`
and `ImageValidatingPolicy` types, explicitly *"aligning with Kubernetes' own trajectory"*
([CNCF announcement](https://www.cncf.io/blog/2025/05/14/announcing-kyverno-release-1-14/),
[CEL libraries](https://kyverno.io/docs/policy-types/cel-libraries/)). (**Unverified**: whether
Kyverno's current CEL engine is cleanly embeddable outside a cluster.) **Verdict: not an option —
but the most direct precedent available for this decision. The one project that built exactly
"Kyverno policies over arbitrary JSON" archived it and moved to CEL. Weigh that.**

**HashiCorp Sentinel** — **Proprietary.** Shipped only inside Enterprise/HCP editions of
Terraform, Vault, Consul and Nomad. No public source, no open-source Go SDK, no license permitting
embedding ([hashicorp.com/sentinel](https://www.hashicorp.com/en/sentinel)). (**Unverified**: the
canonical docs URL 404'd on this pass, so the licensing basis is HashiCorp's marketing pages rather
than a quoted license file.) **Verdict: disqualified for an open-source product.**

**wazero (`github.com/tetratelabs/wazero`) as a generic sandbox** — *"a WebAssembly Core
Specification 1.0 and 2.0 compliant runtime written in Go. It has zero dependencies, and doesn't
rely on CGO."* `Runtime` (`CompileModule()`, `InstantiateModule()`, `NewHostModuleBuilder()`),
`RuntimeConfig`, `ModuleConfig`; an AOT Compiler mode and a platform-agnostic Interpreter mode.
**Memory bounded via `WithMemoryLimitPages()`; termination via `WithCloseOnContextDone()`**, which
*"ensures the executions of functions to be terminated"* on context cancellation
([pkg.go.dev/.../wazero](https://pkg.go.dev/github.com/tetratelabs/wazero)).

**One correction to the brief: wazero does not implement fuel/gas metering.** Its controls are
context cancellation plus a memory page limit, not an instruction budget. That is still the right
shape here, because **a hard memory bound is exactly what OPA and Starlark cannot give you**
(§4.3), and this is the only approach on the list that bounds memory for untrusted policy inside
a single Go binary. Note the natural pairing: OPA bundles can carry a compiled `policy.wasm`
(declared in the manifest's `wasm` field with `entrypoint` and `module`, §3.1), and OPA's own WASM
runtime is already wazero (§2.5) — so "OPA bundles + Rego compiled to WASM + wazero with a memory
limit" is a coherent architecture that keeps the bundle supply chain *and* adds the memory bound.
The cost is the incomplete WASM builtin coverage flagged in §2.5. **Verdict: do not adopt to
start, but this is the documented escape hatch if resource exhaustion becomes a real incident —
and it is worth knowing it exists before designing around the §4.3 gap.**

### 7.6 Comparative verdict

**Does OPA actually win? Yes — but on one axis, and you should be able to say which.**

OPA wins on **policy lifecycle**, and essentially nothing else. Bundles are a complete, boring,
battle-tested answer to the hardest unglamorous part of this product: the format, `roots`-based
multi-tenant scoping, `revision` versioning, JWS signing with file-set integrity, declarative
verification config, persistence for restart, ETag-cached polling, and atomic activate-or-keep-old
semantics. **Re-read the project's own requirement — "policy lives outside the repository it
governs... policy distribution, versioning, integrity and provenance are load-bearing".** OPA is
the only candidate that ships that. CEL, Cedar and Starlark ship none of it.

Secondary strengths: Apache-2.0, CNCF **graduated**, ~monthly releases, a mature embedding API,
first-party `opa test`/`opa fmt`/`opa check --strict`, and comfortable latency (~36 µs median,
§2.1). Rego's termination story is also better than usually credited: **recursion is a compile-time
error** (`rego_recursion_error`), with the explicit rationale that *"policy evaluation should be
known to terminate. Some common programming constructs, like while loops, or recursive references,
can't reasonably provide such guarantees, and are therefore not allowed in Rego"*
([error docs](https://www.openpolicyagent.org/docs/errors/rego-recursion-error/rule-name-is-recursive)).

**Where OPA genuinely loses:**

1. **No cost limit, at all** (§4.3). The only bound is a `context` deadline, which stops the work
   after it has already been done, and bounds nothing about memory. CEL gives a hard pre-flight
   estimate *and* a runtime budget. Against tenant-authored policy this is the most serious gap on
   OPA's side, and §4.3's mitigations are yours to build.
2. **The builtin surface is an attack surface.** `http.send` inside your service means SSRF against
   your cloud metadata endpoint and exfiltration of the diff. The fix (§4.2) is a restricted
   `ast.Capabilities` with the nondeterministic and network builtins stripped — mandatory, easy to
   forget, and it should be a **tested invariant with a denial test**, not a config option. CEL does
   not have this problem because no such builtin exists.
3. **Rego is the hardest language here to learn and review.** Partial rules, unification,
   iteration-by-variable-binding. If tenants are application developers rather than platform
   engineers, expect support load.

**When CEL would be the better call** — if any of these hold:

- **Resource exhaustion by tenant policy is a top-three risk.** Decisive. `CostLimit` plus
  `EstimateCost` plus Kubernetes' already-tuned constants give a defence OPA does not offer.
- **Policies are predominantly boolean predicates over one diff document.** If
  `files.filter(...).size() > N`, prefix matching and globs cover 90% of real policies — and for a
  PR admission controller they very likely do — Rego's extra power is rent you are not collecting.
- **You want a product UI around policy authoring.** The `cel-go/policy` YAML format plus the
  Kubernetes policy/binding/`paramKind` architecture is a far better foundation for a guided editor
  than freeform Rego. Templating a CEL condition into a known-shape YAML policy is tractable;
  generating correct Rego from a UI is not.
- **You want the smallest trust boundary.** No builtin inventory to audit, no capability file to
  maintain.
- **Ecosystem gravity matters.** Kubernetes admission, Envoy and Kyverno 1.14 all run CEL, and
  `kyverno-json` — the one project that tried non-CEL policy over arbitrary JSON — archived itself
  and pointed users at CEL.

**When Go-native predicates would be the better call** — if "tenants author policies" is softer
than stated. If the realistic first-year answer is "we write the policies and ship them", a DSL is
pure overhead: you would build a sandbox, a cost budget, a distribution channel and a signing
pipeline for a problem you do not yet have. And the **config-schema variant satisfies
update-without-redeploy**, which compiled predicates fail — so it is a genuinely viable position,
not just a stepping stone.

**The three defensible positions:**

1. **OPA** — if policy lifecycle is the dominant cost and risk, and you will do the hardening:
   restricted `Capabilities` with `http.send` stripped, per-evaluation deadlines, bounded input,
   bounded `data`, bounded concurrency. **This is the strongest position today, and the bundle and
   signing story alone justifies it.**
2. **CEL** — if hostile-policy resource safety is the dominant risk and policy shapes are
   predicate-shaped. You gain a real cost budget and a smaller trust boundary; you pay in weeks of
   supply-chain plumbing. If you go this way, **copy OPA's `.signatures.json` design rather than
   inventing one.**
3. **A constrained config schema with one expression escape hatch** — the Kubernetes `paramKind`
   pattern. This concentrates untrusted execution into a single narrow path while keeping the
   common 90% bounded by construction, easy to validate, easy to render in a UI and trivially
   signable. It is also the only option that keeps the door open to all three.

**Ruled out:** Cedar (cannot express `count(files under /infra) > 5` — no count, no iteration; no
validator or templates in Go; Lean proofs do not cover the Go port). Starlark for untrusted policy
(*"There is no way to bound the memory used by a Starlark thread"*, with maintainers recommending
a separate process with cgroups). Sentinel (proprietary). `kyverno-json` (archived). JSONLogic
(single-maintainer, no resource bounds, less capable than CEL). Go plugins (ruled out by their own
documentation, four ways over).

---

## 8. Licensing and Project Health

**License: Apache License 2.0**, confirmed from
[`LICENSE`](https://raw.githubusercontent.com/open-policy-agent/opa/v1.21.2/LICENSE) at the pinned
tag. This is the permissive-with-patent-grant license you want for a dependency in an open-source
product: no copyleft obligation on your own code, an express patent grant, and no attribution
burden beyond retaining the notice. There is no dual-licensing, no BSL/SSPL relicensing event in
OPA's history, and no CLA-backed single corporate owner positioned to perform one — see governance
below. **Compatible with any license you pick for the control plane**, including GPL variants.

**CNCF status: Graduated** — the highest maturity tier. From
[cncf.io/projects/open-policy-agent-opa](https://www.cncf.io/projects/open-policy-agent-opa/):
accepted into the CNCF **2018-03-29**, promoted to Incubating **2019-04-02**, and **Graduated
2021-01-29**. Graduated projects are *"considered stable, widely adopted, and production ready"*.

Why that matters beyond a badge: graduation requires a documented, vendor-neutral governance
process, a security disclosure process, and a completed independent security audit. The repository
contains `GOVERNANCE.md`, `MAINTAINERS.md`, `SECURITY.md`, `ADOPTERS.md`, **and
`SECURITY_AUDIT.pdf`** — the audit artifact is checked into the repo. For a project whose thesis is
policy integrity, depending on an engine that has been independently audited and has a published
vulnerability-disclosure process is a material advantage over any single-maintainer alternative in
§7.

**Governance is explicitly designed against single-vendor capture.** From
[`GOVERNANCE.md`](https://raw.githubusercontent.com/open-policy-agent/opa/v1.21.2/GOVERNANCE.md):

> Maintainers use "organizational voting" to approve changes so that no single organization can
> dominate an area of expertise.
>
> Each company or organization (regardless of the number of maintainers associated with or employed
> by that company/organization) receives one organization vote. [...] Individuals not associated
> with or employed by a company or organization are allowed one organization vote.
>
> Unless specified otherwise, a vote passes when greater than fifty percent of the organization
> votes are in favour.

Maintainers are scoped by "area of expertise" — a set of repositories or subtrees — and votes are
counted per *organization*, not per person, specifically so that stacking maintainers from one
employer does not buy control.

**Maintainer distribution**, from
[`MAINTAINERS.md`](https://raw.githubusercontent.com/open-policy-agent/opa/v1.21.2/MAINTAINERS.md)
at v1.21.2 — nine active maintainers across **three** organizations:

| Organization | Maintainers | Areas |
|---|---|---|
| Apple | Anders Eknert, Ash Narkar, Charlie Egan, Stephan Renatus, Tim Hinrichs, Torin Sandall | `opa`, `opa-envoy-plugin`, all repositories |
| Microsoft | Sertaç Özercan, Jaydip Gabani | `gatekeeper`, `gatekeeper-library`, `frameworks/constraints`, `cert-controller`, `gatekeeper-external-data-provider` |
| Google | Max Smythe | `frameworks/constraints`, `gatekeeper`, `gatekeeper-library`, `cert-controller` |

Two honest observations. First, **concentration risk is real**: six of nine maintainers — including
both original creators, Tim Hinrichs and Torin Sandall, who hold "all repositories" — are at Apple.
The cause is documented: on **2025-08-20** OPA's creators announced that *"the creators of Open
Policy Agent (along with many team members from Styra) have joined Apple"*, adding that *"Open
Policy Agent remains a CNCF graduated open source project and there are no changes to the project
governance or licensing"* and that maintainers were unchanged *"except for an organization change
from Styra to Apple"* — which is exactly what `MAINTAINERS.md` reflects. Under organizational
voting Apple still holds one of three votes, so the *governance* protection holds, but the *bus
factor* is concentrated.

A genuinely positive signal accompanied that move: Styra's **commercial** tooling is being
contributed to the CNCF OPA organization — EOPA (their commercial OPA distribution), **OPA Control
Plane** (§6.8, bundle building and distribution — i.e. this project's §3 as a product), several
SDKs, and **Regal**, a Rego linter worth knowing about for policy-author tooling. The commercial
moat is being donated rather than withdrawn, which is the opposite of the usual
acquisition outcome. (**Unverified / do not record as fact**: claims circulating that Styra "shut
down" on a particular date trace to a competitor's migration-pitch blog. What is verifiable is the
Apple announcement above, and that `styra.com` and `docs.styra.com` no longer resolve as of
2026-10-13 while `openpolicyagent.org` resolves normally.) Second, and reassuringly, **Gatekeeper and the constraint framework are maintained in
the same org by the Microsoft and Google maintainers** — which is why §6's prior art is not a
third-party add-on but a sibling project under shared governance. That is a genuine signal for
reusing their design.

The emeritus list (eight names) indicates a maintainer rotation process that actually functions
rather than a static founding group.

**Release cadence — reliably monthly minors with patch releases between.** Verified from the
GitHub releases API:

| Release | Date | | Release | Date |
|---|---|---|---|---|
| v1.21.2 | 2026-10-12 | | v1.16.0 | 2026-04-30 |
| v1.21.1 | 2026-09-29 | | v1.15.2 | 2026-04-08 |
| v1.21.0 | 2026-09-24 | | v1.15.0 | 2026-03-26 |
| v1.20.0 | 2026-08-27 | | v1.14.0 | 2026-02-26 |
| v1.19.0 | 2026-07-30 | | v1.13.0 | 2026-01-29 |
| v1.18.0 | 2026-06-25 | | v1.12.3 | 2026-01-14 |
| v1.17.0 | 2026-05-28 | | | |

Nine minor releases in roughly nine months, each with one or two patch follow-ups. **The patch
releases are mostly security-driven, which is the behaviour you want:** v1.21.2 exists to rebuild
binaries against Go v1.27.2 and `golang.org/x/net` v0.60.0 *"to address security vulnerabilities in
Go's standard library and networking packages"*, plus one correctness fix (encoded slashes handled
differently between basic-auth policy input and the Data API, issue #9323). A project that cuts a
release to pick up an upstream Go CVE within days is actively maintained in the way that matters.

Practical consequences for depending on this long-term:

- **Budget for monthly dependency bumps.** At ~monthly minors, pinning for a year means being ~9
  minor versions behind, which makes the eventual upgrade a project rather than a bump. Wire OPA
  into automated dependency updates with your policy test corpus as the gate.
- **Go version coupling is tight.** `go.mod` at v1.21.2 declares `go 1.26.0`, and the project's own
  example code uses `errors.AsType[ast.Errors](err)` — a **Go 1.26** addition. OPA tracks recent Go
  closely, so this project's own minimum Go version is effectively set by OPA's. Note that the
  code in §4.4 deliberately uses `errors.As` rather than `errors.AsType` to avoid requiring the
  newest toolchain.
- **The v0 → v1 transition is the one real API break to know about**
  ([v0 compatibility](https://www.openpolicyagent.org/docs/v0-compatibility)). It moved the Go
  import paths (§"Read This First"), left the old paths as compatibility wrappers, and changed the
  *language* default: `if`, `contains`, `in` and `every` became mandatory keywords rather than
  opt-in via `import rego.v1`/`future.keywords.*`. Version selection is available three ways:
  `Module.SetRegoVersion()` and `Bundle.Manifest.SetRegoVersion()` per module/bundle, the
  `rego.SetRegoVersion` option at evaluation time with `ast.RegoV0`/`ast.RegoV1`, and
  `sdk.Options.V0Compatible`/`V1Compatible`. `opa fmt --v0-v1` and `opa check --v0-v1` assist
  migration. **Since v1.0 (January 2025) the API has been stable**; the v1.x series has added
  options without breaking existing ones.
- **Relicensing risk is low but not zero.** Apache-2.0 plus CNCF graduation plus three-organization
  governance plus a vendor-neutral voting rule is about as much structural protection as an
  open-source dependency offers. The residual concern is maintainer concentration at a single
  employer, not the license.

A final health signal worth stating: **Gatekeeper, Conftest, and the constraint framework all live
in the same `open-policy-agent` GitHub organization under the same governance.** Section 6's prior
art is therefore not "some projects that happen to use OPA" — it is a family of sibling projects
with shared maintainers. Designs borrowed from Gatekeeper are unlikely to drift away from OPA's
own direction.

---

## 9. Recommendations, Condensed

1. **Import `github.com/open-policy-agent/opa/v1/...`, never the legacy paths.** Pin to v1.21.2 or
   later and automate the monthly bump with a policy test corpus as the gate.
2. **Use the `sdk` package, not bare `rego`.** Bundles, signing, and decision logs are the
   project's load-bearing requirements and they live only there (§1.6). Keep `rego` for a local
   dry-run path.
3. **Compile once, evaluate many**, behind an `atomic.Pointer[rego.PreparedEvalQuery]` if you use
   the low-level API. Never call `rego.New` on the request path.
4. **Treat undefined as a third state.** `len(resultSet) == 0` (or `sdk.IsUndefinedErr`) is not
   `false`. Then use the **three-way** failure split from §6.1, not a two-way one: unreachable →
   allow; evaluation error → deny; self-exemption attempt → always deny. Make the fail-open/
   fail-closed choice a **per-policy declaration**, not two hard-coded call sites.
4b. **Make severity a per-policy, per-enforcement-point field** — Gatekeeper's `dryrun`/`warn`/
   `deny` plus `scoped`/`scopedEnforcementActions` (§6.1). Without graduated rollout the product is
   not adoptable, and this is also how an expensive policy gets demoted to the asynchronous audit
   path instead of being dropped.
5. **Set `rego.Capabilities` to an allowlist with `http.send` and all nondeterministic builtins
   removed, and `AllowNet: []string{}`.** Make it a tested invariant with a denial test, not a
   config option. This closes the §4.2 bypass and makes decisions reproducible for audit.
6. **Bound the input before evaluating** — file count, patch size, path depth, total bytes. Reject,
   never truncate. This is the highest-value mitigation for the §4.3 memory gap, which OPA does not
   close for you.
7. **Set `GOMEMLIMIT` and bound evaluation concurrency.** 21 KB and ~382 allocations per decision
   times unbounded goroutines is how this service dies.
8. **Enable `rego.StrictBuiltinErrors(true)` and `rego.Strict(true)`.** The default turns a broken
   policy into a permissive one.
9. **Ship policy as signed snapshot bundles from an OCI registry**, built with
   `opa build -b . -O=1 -r "$GIT_SHA"`, signed in a separate privileged job, with `persist: true`,
   per-tenant `roots`, and distinct signing `scope`s for the fail-closed and fail-open policy
   families (§3.6).
10. **Alert on bundle activation failure.** A failed signature leaves the service silently
    enforcing stale policy — this is the system's most important operational alert (§3.4).
11. **Log decisions with `console: true` to start**, mask anything derived from diff content, and
    record `Bundles[].Revision` so input + revision + result is a reproducible audit triple (§5.1).
12. **Copy Gatekeeper's two-level template/constraint split**, with a schema on the parameters and
    **binding failures rejected at submit time** — a constraint that fails to bind must never
    degrade to a silent no-op (§6.1). Add **audit mode** (§6.1): it is the rollout-safety number,
    the drift detector, and what makes failing open defensible. And copy Kubernetes' CEL cost
    constants as a latency-budget sanity check even though you are not using CEL (§7.1).
12b. **Define a versioned diff-JSON schema as the policy input contract** — per-object change
    records with a closed action enum and explicit `sensitive`/`unknown` masks, `input.diff` and
    `input.pr` as siblings, and GitHub-to-schema as a separate bounded layer (§6.4). Nobody in the
    OPA ecosystem has published one, so this is both the project's main design surface and
    unoccupied ground (§6.7).
12c. **Exempt your own repositories and build a break-glass path outside the enforced system before
    you need it** (§6.1) — and check that your deployment automation will not immediately undo it.
12d. **Read OPA Control Plane** (§6.8) before finalising §3. It is this project's distribution
    thesis already implemented by OPA's own authors, and it names a problem you will otherwise hit
    late: what happens when a global policy and a tenant policy disagree.
13. **Skip WASM and IR.** You have four orders of magnitude of latency headroom; use `opa build
    -O=1` if you ever need more (§2.5).
14. **Revisit the engine choice if either premise changes:** if tenants become mutually hostile,
    §4.3 forces process or WASM isolation regardless of engine; if "tenants author policy" softens,
    a constrained config schema beats every option here (§7.6).
