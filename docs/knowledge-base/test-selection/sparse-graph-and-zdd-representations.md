# Sparse graph and ZDD representations

*Test selection · Authored for this project — advisory. Sources: Knuth, TAOCP Vol. 4A §7.1.4 (BDDs and ZDDs); Minato 1993 (zero-suppressed BDDs); compressed sparse row and column formats from the sparse-matrix literature*

## What it is

There are three ways to hold the sparse relations test selection reads: item → tests
(coverage) and package → importers (impact).

- **CSR** (compressed sparse row) stores rows as an `offsets` array plus a flat `targets` array.
  Row `r` is `targets[offsets[r]:offsets[r+1]]`. Its transpose, **CSC**, answers the reverse
  question: test → items.
- **Reverse-dependency reachability** is a breadth-first search over the importer graph in CSR,
  costing `O(V+E)` per query.
- A **ZDD** (zero-suppressed decision diagram) is a canonical DAG for a *family of sets*. A node
  whose 1-branch leads to the empty sink ⊥ is removed, so sparse families share structure and
  can be exponentially smaller than a list of their members.

## Why it exists

Maps of slices are pointer-heavy, scatter memory, and make the transpose a second
hand-maintained copy that drifts. CSR and CSC are two flat arrays each, built once per upload
in `O(nnz)` after sorting, and safe to share read-only across plans. Impact needs **every**
transitive importer of a changed package. Precomputing the full closure costs `O(V²/w)` words
(20,000 packages → ~50 MB) and has to be redone on every upload. A BFS per plan touches only
what is reachable.

## When to use it

- **CSR/CSC** for the coverage index and the dependency graph, built when CI uploads them (C-12,
  v0.0.2 C-14) and replaced whole, never patched.
- **BFS over reverse edges** for impact, bounded by v0.0.2 NFR-14 (≤ 20,000 packages, ≤ 200,000
  edges, 250 ms).
- **ZDDs** only when the question is about *families*: "every historical trace containing both
  `a` and `b`", or set algebra over millions of traces with heavy sharing. That is not this
  project's question. A ZDD also needs a library (Go's standard library has none), and
  rebuilding or applying operations on every commit costs more than a CSR rebuild.

| Representation | Build | Query | Update on a new commit | Space |
|---|---|---|---|---|
| CSR / CSC | `O(nnz log nnz)` sort, then `O(nnz)` | row: `O(deg)`; transpose read: `O(deg)` | rebuild `O(nnz log nnz)` | `4·(V+1) + 4·nnz` bytes |
| Map of slices | `O(nnz)` | `O(deg)` + hashing | in place, but the transpose drifts | pointers + headers per row |
| Full transitive closure (bitsets) | `O(V·(V+E))` | `O(1)` per pair | rebuild | `V²/8` bytes |
| ZDD of a trace family | depends on variable order; can be exponential in the worst case | `O(size)` per family operation | apply ops, memoised | shared nodes; can be far below the member count |

**The OPA boundary.** Reachability is computed in Go. `ci.diff.v1` carries only the bounded
result: impacted packages, capped by NFR-7, or a `plan_too_large` trigger. Rego never walks a
graph of this size.

## How to use it

```go
// RIGHT — reverse importer graph in CSR; BFS marks every transitive importer once.
type Graph struct {
	offsets []uint32
	targets []uint32
}

func (graph Graph) ImportersOf(changed []uint32) []bool {
	reached := make([]bool, len(graph.offsets)-1)
	queue := slices.Clone(changed)
	for _, node := range changed {
		reached[node] = true
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, importer := range graph.targets[graph.offsets[node]:graph.offsets[node+1]] {
			if !reached[importer] {
				reached[importer] = true
				queue = append(queue, importer)
			}
		}
	}
	return reached
}
```

**Invariant:** `offsets` has length `V+1`, is non-decreasing, and ends at `len(targets)`. Every
target is `< V`. The parser checks this once, at upload, and refuses the graph otherwise.

## How not to use it

```go
// WRONG — recursion without a visited set.
func importers(pkg string) []string {
	var all []string
	for _, imp := range reverse[pkg] {
		all = append(all, append([]string{imp}, importers(imp)...)...)
	}
	return all
}
```
Symptom: exponential time on diamond-shaped graphs. Go's compiler forbids import cycles, test
packages included, but an uploaded graph is untrusted input: a malformed one with a cycle
recurses until the stack is gone. Instead, use an iterative BFS with a visited set.

```go
// WRONG — the closure precomputed on every upload, for every pair.
closure := transitiveClosure(graph)
```
Symptom: memory grows with `V²`, and every upload pays for pairs no plan will ask about.
Instead, run a BFS per plan from `ΔC` only.

```go
// WRONG — a ZDD library for a few hundred tests.
family := zdd.FromTraces(traces)
```
Symptom: a dependency and a variable-ordering problem in exchange for no measurable gain.
Instead, use CSR plus bitsets, and adopt a ZDD only when traces number in the millions with
proven sharing.

Three more failure modes:
- **Cold start:** with no graph, impact is unknown, which forces the full matrix, never "no
  importers".
- **Deleted packages:** resolve them in the base commit's graph.
- **Non-Go changes:** these have no node, so they are triggers.

**Trade-off:** CSR is immutable. Every upload rebuilds it whole, which is the price of a
representation that is impossible to leave half-updated.

**Related:** `bitwise-coverage-matrices.md`, `digital-search-trees-for-path-matching.md`.
