# Bitwise coverage matrices

*Test selection · Authored for this project — advisory. Sources: Knuth, TAOCP Vol. 4A §7.1.3; Chambi, Lemire et al. 2016 ([arXiv:1402.6407](https://arxiv.org/abs/1402.6407)); Lemire et al. 2018 ([arXiv:1709.07821](https://arxiv.org/abs/1709.07821)); Go `math/bits`*

## What it is

Coverage is a bipartite relation `G = (C ∪ T, E)`: code items `C` (files, or line blocks) and
individual tests `T`, with an edge when a test executed the item. Give every test a dense index
`0 … |T|−1`; the row for an item is then a bitvector of `⌈|T|/w⌉` words (`w = 64`). Test
**selection** for a change `ΔC` is the OR of `K = |ΔC|` rows. The broadword operations of TAOCP
§7.1.3 work on the rows a word at a time: sideways addition (`bits.OnesCount64`), the rightmost
1-bit (`bits.TrailingZeros64`), and clearing it (`x & (x-1)`) or a mask (`x &^ y`).

## Why it exists

Item-to-test maps held as `map[string][]string` turn every plan into string hashing,
allocation and de-duplication. The result costs `O(Σ postings)` allocations and is easy to get
wrong on duplicates. As words, union is `O(K·|T|/w)` word operations with no allocation beyond
the result, and counting and iteration are a few instructions per word.

## When to use it

- The plan needs union, intersection, difference or counts over test sets, which is every plan.
- Rows are built **per plan, only for `ΔC`**. `K` is bounded by NFR-2 (≤ 1,000 changed files),
  so the dense working set is at most `K·|T|/64` words.
- The **repository-wide** index (every item → its tests) is stored sparse. Use sorted `[]uint32`
  postings while `|T|` is under about 65,536. Use Roaring when `|T|` is larger and rows are
  clustered: its 2¹⁶ chunks hold a sorted `uint16` array up to 4,096 values, an 8 KiB bitmap
  above that, or runs of consecutive values.
- This repository has a few hundred tests, so dense rows plus sorted postings suffice. Roaring
  (`github.com/RoaringBitmap/roaring`) is a dependency to adopt on evidence, not by default.

| Structure | Build | Union over `K` rows | Update one item | Space |
|---|---|---|---|---|
| Dense rows for all of `C` | `O(nnz + |C|·|T|/w)` | `O(K·|T|/w)` | `O(|T|/w)` | `|C|·|T|/w` words |
| Sorted postings (CSR) | `O(nnz log nnz)` | `O(Σ postings + |T|/w)` | `O(postings)` | `4·nnz` bytes |
| Roaring | `O(nnz)` | `O(Σ containers)`, ≤ 1,024 words per bitmap container | `O(log chunks + container)` | ≈ 2 B per value in arrays, 8 KiB per dense chunk |

**The OPA boundary.** Rego has no word operations, and NFR-6 gives an evaluation 250 ms. The
matrix stays in Go. Only the bounded projection crosses into `ci.diff.v1`: the selected test
names (NFR-7: ≤ 512) and the counts.

## How to use it

```go
// RIGHT — rows share one test index; union and iteration are word operations.
type TestSet struct {
	index IndexRevision
	words []uint64
}

func (set TestSet) Union(other TestSet) (TestSet, error) {
	if set.index != other.index {
		return TestSet{}, ErrMixedIndex
	}
	joined := slices.Clone(set.words)
	for i, word := range other.words {
		joined[i] |= word
	}
	return TestSet{index: set.index, words: joined}, nil
}

func (set TestSet) Size() (count int) {
	for _, word := range set.words {
		count += bits.OnesCount64(word)
	}
	return count
}

func (set TestSet) Each(visit func(test int)) {
	for i, word := range set.words {
		for ; word != 0; word &= word - 1 {
			visit(i*64 + bits.TrailingZeros64(word))
		}
	}
}
```

**Invariant:** every row in one computation comes from the same test index, assigned once per
coverage revision. A union across two revisions is refused, never computed.

## How not to use it

```go
// WRONG — an unmapped item reads as "covered by no test".
row, _ := index.RowOf(file)
selected, _ = selected.Union(row)
```
Symptom: a new or uncovered file selects nothing and its tests never run. Instead, return an
explicit miss, which forces the full matrix with `unmapped_file` (v0.0.1 FR-22).

```go
// WRONG — tests added by the change are not in the index yet, so they are skipped.
plan := index.Select(changedFiles)
```
Symptom: a new `TestX` in a changed `_test.go` never runs until coverage is re-uploaded.
Instead, always add tests the diff declares or edits, and treat a cold index as the full matrix.

```go
// WRONG — a deleted file has no row at head, so its dependants are not selected.
row, ok := indexAtHead.RowOf(deleted)
```
Symptom: removing a function breaks callers' tests, and none of them run. Instead, look a
deleted or renamed file up in the index built at the **base** commit.

```go
// WRONG — bit-by-bit iteration: 64 branches per word, most of them empty.
for i := 0; i < 64*len(set.words); i++ {
	if set.words[i/64]>>(i%64)&1 == 1 {
		visit(i)
	}
}
```
Symptom: iteration cost scales with `|T|`, not with the selected count. Instead, use `Each`
above: one `TrailingZeros64` per set bit.

Two more failure modes:
- **Configuration edits:** non-code changes (lockfiles, CI config) have no row by nature. They
  are triggers, not lookups.
- **Flaky coverage:** coverage that differs between runs is merged by **union across runs**,
  never by keeping the last run.

**Trade-off:** dense rows cost `|T|/8` bytes per changed item. That is trivial at `K ≤ 1,000`,
but it is why the repository-wide index stays sparse.

**Related:** `test-suite-minimization-and-set-cover.md`, `sparse-graph-and-zdd-representations.md`.
