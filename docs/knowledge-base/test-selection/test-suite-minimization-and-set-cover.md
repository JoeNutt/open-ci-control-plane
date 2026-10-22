# Test suite minimisation and set cover

*Test selection · Authored for this project — advisory. Sources: Knuth, TAOCP Vol. 4B §7.2.2.1 and Vol. 1 §2.2.6; Johnson 1974 and Chvátal 1979 (greedy bound); Feige 1998 and Dinur–Steurer 2014 (hardness); Wong et al., ICSE 1995; Rothermel et al., ICSM 1998 and STVR 2002*

## What it is

**Selection** is the union of every test reaching a changed item: `S = ⋃_{c∈ΔC} T(c)`, costing
`O(K·|T|/w)` as bitvector ORs. **Minimisation** keeps a subset `T' ⊆ S` that still covers every
coverable changed item, either the fewest tests or the least recorded duration. That is
minimum (weighted) set cover, NP-hard, and its dual is hitting set. The greedy heuristic picks
the test with the best `newly covered / cost` ratio until nothing coverable remains. Its cost
is within `H(K) ≤ ln K + 1` of optimal (Johnson; Chvátal for weights). No polynomial algorithm
does better than `(1−ε)·ln K` unless P = NP (Dinur–Steurer, after Feige).

## Why it exists

Selection on a widely used file selects nearly everything, so it saves nothing on exactly the
changes that are slowest. Minimisation removes tests that are redundant **for the coverage
measure**. Whether that also preserves fault detection is **contested**: Wong et al. (1995)
reported little loss at constant coverage, while Rothermel et al. (1998, 2002) reported severe
loss on seeded faults. Minimisation is therefore always shadowed before it is enforced.

## When to use it

- Selection is already proven safe in shadow; minimisation is a second, separately gated step.
- Coverage is at **line or block** granularity per individual test. At file granularity,
  "covers the change" is too coarse to minimise against.
- Use **greedy** in the request path: `O(rounds · |S| · K/w)`, rounds ≤ `min(K, |S|)`, under a
  deadline (v0.0.2 NFR-16: 250 ms) with selection as the fallback.
- Use an **exact** search only offline, or when `|S|` is tiny (≈ ≤ 25 candidates), to measure how
  far greedy lands from optimal. Knuth's DLX (Algorithm X over toroidal doubly linked lists,
  §7.2.2.1; the list mechanics are §2.2.6) solves **exact** cover, where every item is covered
  exactly once. Set cover lets an item be covered more than once, so it needs §7.2.2.1's
  multiplicity variant (Algorithm M) or a bitset branch-and-bound. Neither bounds the worst
  case below exponential.

| Method | Time | Space | Guarantee |
|---|---|---|---|
| Selection (union) | `O(K·|T|/w)` | `|T|/w` words | Every reaching test |
| Greedy weighted cover over bitsets | `O(min(K,|S|)·|S|·K/w)` | `|S|·K/w` words | ≤ `H(K)`·optimal |
| Exact (DLX / branch-and-bound) | `O(2^|S|)` worst case | `O(nnz)` links | Optimal |

## How to use it

```go
// RIGHT — greedy weighted cover; each candidate's row is over the K changed items.
func Minimised(rows []CoverRow, coverable ItemSet) ([]TestID, error) {
	uncovered := coverable.Clone()
	var chosen []TestID
	for !uncovered.Empty() {
		best, bestRatio := -1, 0.0
		for i, row := range rows {
			gain := row.items.IntersectionCount(uncovered)
			if gain == 0 {
				continue
			}
			if ratio := float64(gain) / row.seconds; ratio > bestRatio {
				best, bestRatio = i, ratio
			}
		}
		if best < 0 {
			return nil, ErrUncoverableItems
		}
		chosen = append(chosen, rows[best].test)
		uncovered = uncovered.Without(rows[best].items)
	}
	return chosen, nil
}
```

**Invariant:** the union of the chosen rows covers every coverable changed item, and the caller
checks this before dispatch. Items no test covers never enter `coverable`; they force the
unminimised selection, never a smaller plan.

## How not to use it

```go
// WRONG — "coverable" includes items no test reaches, so the loop can never finish.
uncovered := changedItems
```
Symptom: the deadline fires on every change touching new code. Instead, intersect with the
union of candidate rows first, and report the remainder as uncovered.

```go
// WRONG — exact cover: a test overlapping another's items is rejected outright.
solution := dlx.Solve(matrix)
```
Symptom: no solution, or a far larger one, whenever two tests share a line, which is almost
always. Instead, use set cover (greedy, or multiplicities with branch-and-bound).

```go
// WRONG — an unbounded exact search on the webhook path.
best := exactMinimumCover(rows)
```
Symptom: plans time out on large changes. Instead, use greedy under a deadline, and keep exact
search for offline measurement.

```go
// WRONG — a test with no recorded duration gets cost 0, so its ratio is +Inf.
ratio := float64(gain) / row.seconds
```
Symptom: an unmeasured slow test always wins. Instead, refuse to minimise without durations
(v0.0.2 FR-47), or use a stated default cost and label the plan with it.

Four more failure modes:
- **New tests:** a test with no coverage yet is never minimised away; it always runs.
- **Deleted items:** they leave `ΔC`, but the tests that reached them at the base commit stay
  candidates.
- **Configuration edits:** these are full-matrix triggers before any minimisation.
- **Flaky coverage:** union across runs, as in selection.

**Trade-off:** each test removed saves time and loses an unknown amount of fault detection,
which is why it is gated by shadow escapes rather than by trust.

**Related:** `bitwise-coverage-matrices.md`, `sparse-graph-and-zdd-representations.md`.
