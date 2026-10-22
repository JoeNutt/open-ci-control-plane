# Input validation

*Security · Derived from the global `sec-input-validation` skill — **binding on this project***

## What it is

**All input is evil until proven otherwise** — not just form fields, but headers, cookies,
query strings, path segments, uploaded file contents *and* names, webhook payloads, API
responses from services you do not control, environment variables, and database rows written
by something else. Define what is **valid** and reject everything else; **canonicalise before
validating**; bound every resource before the expensive work.

## Why it exists

Because denylists lose. The list of bad inputs is unbounded, encodings multiply it, and you
will be outnumbered — `<script`, `<SCRIPT`, `<scr\0ipt`, `%3Cscript`, `<svg onload` all reach
the same place.

And because the single most common validation bypass is checking a string before reducing it
to its one true form. The same resource has endlessly many spellings, so a validator that
sees a different form from the consumer is a hole.

Client-side validation is a usability feature. **The attacker does not use your client.**

## When to use it

At every entry point, for every field: decide and enforce type, length or range, format, and
character set. Then **reject — do not clean.** Stripping characters produces new inputs you
did not anticipate (removing `../` from `....//` yields `../`) and hides attacks from your
logs.

Order matters and is not negotiable: **decode → canonicalise → validate → use.** Decode
exactly once, and reject input still encoded afterwards rather than looping.

## How to use it

```go
// RIGHT — allowlist, bounded, and the result is a domain type.
const (
    maxRepositorySegment  = 100
    repositoryPunctuation = "._-"
)

func parseRepository(repository string) (Repository, error) {
    if len(repository) > 2*maxRepositorySegment+1 {
        return Repository{}, ErrMalformedRepository      // bound before the walk
    }
    owner, name, separated := strings.Cut(repository, "/")
    if !separated || !repositorySegmentOK(owner) || !repositorySegmentOK(name) {
        return Repository{}, ErrMalformedRepository      // reject, do not clean
    }
    return Repository{Owner: owner, Name: name}, nil
}

func repositorySegmentOK(segment string) bool {
    if len(segment) == 0 || len(segment) > maxRepositorySegment {
        return false
    }
    if !asciiAlphanumeric(segment[0]) {
        return false
    }
    for index := 1; index < len(segment); index++ {
        if !repositoryCharacterOK(segment[index]) {
            return false
        }
    }
    return true
}

func repositoryCharacterOK(character byte) bool {
    return asciiAlphanumeric(character) ||
        strings.IndexByte(repositoryPunctuation, character) >= 0
}
```

The allowlist is **named predicates, not a pattern**. Byte comparison is sound where the
accepted set is ASCII-only: every byte of a multi-byte rune fails the predicates, exactly as a
character class would reject the rune. `golang.org/x/mod/module` validates module and import
paths this way and imports no `regexp` at all — `checkElem` walks runes through `modPathOK` and
`importPathOK`, which do range comparisons and a `strings.ContainsRune` against a constant.

```go
// RIGHT — never build a path from input. Resolve, then verify containment.
func resolveInBundle(root, rel string) (string, error) {
    abs := filepath.Join(root, filepath.Clean("/"+rel))   // clean, then join
    abs, err := filepath.EvalSymlinks(abs)                // a symlink inside can point out
    if err != nil {
        return "", err
    }
    if !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
        return "", ErrOutsideBundle                        // compare RESOLVED paths
    }
    return abs, nil
}
```

**The invariant: every resource has an explicit limit enforced before the expensive work.**
Body size, header size and count, array lengths, page size (cap it — never trust the client's),
upload size *and decompressed size and entry count*, image dimensions before decoding,
nesting depth, timeouts on every call, concurrency and rate limits.

**Fail closed.** When a limit is hit, reject with a clear error — **never truncate silently,
because truncation turns a rejected input into an accepted, different one.**

## How not to use it

**A denylist.**

```go
// WRONG — enumerating the bad.
if strings.Contains(input, "../") || strings.Contains(input, "<script") {
    return ErrBad
}
// Symptom: `....//`, `%2e%2e%2f`, `<SCRIPT`, `<svg onload` all pass.
```

**Cleaning instead of rejecting.**

```go
// WRONG — stripping produces a new input nobody validated.
clean := strings.ReplaceAll(path, "../", "")
// Symptom: "....//etc/passwd" becomes "../etc/passwd". The attack also no longer
// appears in the logs, because it looks like a legitimate request.
```

**Validating before canonicalising.**

```go
// WRONG — the check sees one spelling, the consumer sees another.
if !strings.HasPrefix(userPath, "/bundles/") { return ErrBad }
f, _ := os.Open(filepath.Clean(userPath))      // Clean happens AFTER the check
// Symptom: "/bundles/../../etc/passwd" passes the prefix test and resolves outside.
```

**Bounding after the work.**

```go
// WRONG — the body is fully read before its size is considered.
body, _ := io.ReadAll(r.Body)
if len(body) > maxBodyBytes { return ErrTooLarge }
// Symptom: memory is already allocated. Use http.MaxBytesReader, and check the
// declared size of an archive before decompressing rather than after.
```

**A regex where a predicate would read better.**

```go
// WRONG — the accepted set is assembled at run time and stated nowhere a reader
// can see it.
var segment = fmt.Sprintf(`[A-Za-z0-9][A-Za-z0-9._-]{0,%d}`, maxSegment-1)
var repoPattern = regexp.MustCompile("^" + segment + "/" + segment + "$")
// Symptom: to learn what a repository is, a reader parses a pattern built by
// Sprintf. The rule cannot be unit tested on its own, two segments with
// genuinely different rules cannot be expressed without making it unreadable,
// and the off-by-one between {0,99} and a hundred-byte bound is invisible.
```

Prefer a named predicate per accepted set. Reach for a pattern when the grammar is genuinely
irregular — nested optional groups, alternation, backreferences — not merely because the input
is a string.

**An unbounded regex over untrusted input.** Where a pattern is the right tool, catastrophic
backtracking on crafted input is a denial of service. A regex over untrusted input should be
linear-time or time-bounded. Go's `regexp` is RE2 and linear by construction; most other
engines are not.

**Comparing unnormalised text for a security decision.** Normalise (NFC/NFKC) **before**
validating, not after — a normalisation applied downstream can reintroduce a character your
validator rejected. Use invariant comparison, not a locale-aware `ToLower`.

**Trade-off:** strict allowlists reject legitimate input at the edges — unusual names, unusual
encodings — and each rejection is a support question. That is cheaper than the alternative.

**Related:** `../engineering/defensive-design.md`, `injection-defense.md`, `threat-modeling.md`
