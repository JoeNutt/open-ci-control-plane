# Test selection

The algorithms and data structures behind test impact analysis and test-suite minimisation. One
file per subject. Every entry is **advisory**: authored for this project from cited sources,
with contested claims attributed.

| Entry | Read it for |
|---|---|
| `bitwise-coverage-matrices.md` | The coverage graph as bitvectors over individual tests; selection as a union; broadword operations in `math/bits`; when Roaring earns its dependency |
| `test-suite-minimization-and-set-cover.md` | Selection against minimisation; greedy weighted set cover and its `H(K)` bound; why DLX solves exact cover, not set cover; the contested fault-detection evidence |
| `sparse-graph-and-zdd-representations.md` | CSR/CSC for coverage and imports; reverse-dependency reachability; when a ZDD is and is not warranted |
| `digital-search-trees-for-path-matching.md` | Segment tries and Patricia trees for longest-prefix directory resolution; why general globs stay off them |

The unit throughout is the **individual test**, with coverage per test, not the Go package. In
Go, per-test coverage comes from building with `-covermode=atomic -coverpkg=./...` and calling
`runtime/coverage.ClearCounters()` and `WriteCountersDir()` around each test, which Go's
documentation says needs atomic counters.

Heavy computation happens in Go before OPA evaluates anything. The policy input document
carries only bounded results.
