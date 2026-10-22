# Digital search trees for path matching

*Test selection · Authored for this project — advisory. Sources: Knuth, TAOCP Vol. 3 §6.3 (digital searching: tries, Patricia); Morrison 1968 (Patricia)*

## What it is

A **trie** keyed by path **segments**: each edge is one directory name, and a node may carry the
group or suite registered for that directory. Resolving a changed path walks its segments from
the root and keeps the deepest registered node passed (longest prefix wins). A **Patricia** /
**radix** tree compresses each chain of single-child nodes into one edge, so depth tracks the
branching points rather than the segment count. Each path is resolved in `O(L)` (`L` = segments,
≤ 20 by NFR-2), not by testing every rule.

## Why it exists

The naive resolver tests each of `M` rules against each of `K` changed paths: `O(K·M·L)`. At
NFR-2 and NFR-4 bounds (`K ≤ 1,000`, `M ≤ 512` path patterns) that is about 10 million segment
comparisons per plan. A trie makes it `O(K·L)`, independent of `M`, and gives "longest prefix
wins" by construction instead of by sorting rules and hoping no two tie.

## When to use it

- The rules are **directory prefixes**. Binding globs already are: `parseBoundGlob` admits only
  `<directory>/**`. That holds for v0.0.1's binding paths and for suite-to-directory resolution.
- `M` is large enough to matter. With this repository's one binding of under 20 globs, the
  linear loop costs microseconds. The trie is worth building once rules or repositories grow,
  or once the same index is consulted for every changed path **and** every LCOV `SF:` path.
- **Not** for general globs (`*` mid-segment, `**` mid-path). Those are not prefix languages and
  stay on the linear matcher, or need an automaton, which is a different entry.

| Approach | Build | Resolve `K` paths | Space |
|---|---|---|---|
| Linear rule loop | `O(1)` | `O(K·M·L)` | `O(M)` |
| Segment trie | `O(M·L)` | `O(K·L)` | `O(Σ segments)` nodes |
| Patricia (compressed) | `O(M·L)` | `O(K·L)`, fewer node hops | `O(M)` branching nodes |

**The OPA boundary.** Rego's `glob.match` per rule per path is the naive loop, inside a 250 ms
deadline. Group placement is resolved in Go, so `ci.diff.v1` carries each changed file's
resolved group: bounded, and computed once.

## How to use it

```go
// RIGHT — segment trie; the deepest registered directory on the walk wins.
type DirectoryTrie struct {
	children map[string]*DirectoryTrie
	group    GroupName
	isOwned  bool
}

func (trie *DirectoryTrie) Register(directorySegments []string, group GroupName) {
	node := trie
	for _, segment := range directorySegments {
		next, found := node.children[segment]
		if !found {
			next = &DirectoryTrie{children: map[string]*DirectoryTrie{}}
			node.children[segment] = next
		}
		node = next
	}
	node.group, node.isOwned = group, true
}

func (trie *DirectoryTrie) GroupOf(path RepositoryPath) (GroupName, bool) {
	node, deepest, found := trie, GroupName(""), false
	for _, segment := range path.Segments() {
		if node = node.children[segment]; node == nil {
			break
		}
		if node.isOwned {
			deepest, found = node.group, true
		}
	}
	return deepest, found
}
```

**Invariant:** keys are whole segments of a parsed `RepositoryPath` (`Segments()` exists) or
`RepositoryDirectory` (it has no `Segments()` yet; add one beside `String()` rather than
splitting its text a second time), never raw substrings. Registering one directory to two groups is refused at parse time (v0.0.1
binding rule: "a path in two groups").

## How not to use it

```go
// WRONG — a byte prefix, not a segment prefix.
if strings.HasPrefix(path, "internal/change/domain") {
	return domainGroup
}
```
Symptom: `internal/change/domainx/file.go` is placed in `domain`. Instead, match whole segments
(a trie keyed by segment does this by construction).

```go
// WRONG — the first matching rule wins, in map iteration order.
for glob, group := range rules {
	if glob.Matches(path) {
		return group
	}
}
```
Symptom: with nested directories registered, the group changes between runs. Instead, take the
longest prefix, deterministically.

```go
// WRONG — a general glob squeezed into the trie by stripping its wildcards.
trie.Register(strings.TrimSuffix("internal/*/testdata/**", "/**"), fixtures)
```
Symptom: the literal segment `*` never matches anything, and those files silently fall into no
group. Instead, keep non-prefix patterns on the linear matcher, and refuse them where only
prefixes are allowed (as `parseBoundGlob` does).

Three more failure modes:
- **Deleted files:** these resolve the same way; the path still names its directory.
- **New directories:** these resolve to their nearest registered ancestor, or to no group. In
  the direction rule, "no group" is never refused (v0.0.1 §13).
- **Configuration files:** resolution places them; classification (lockfile, CI config) is a
  separate rule set with general globs.

**Trade-off:** a trie is a second representation of the binding, so it must be built from the
parsed binding in one place and never edited independently.

**Related:** `sparse-graph-and-zdd-representations.md`, `bitwise-coverage-matrices.md`.
