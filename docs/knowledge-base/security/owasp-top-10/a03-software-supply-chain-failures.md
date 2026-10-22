# A03:2025 — Software Supply Chain Failures

*OWASP Top 10:2025 · External: OWASP*

## What it is

**New in 2025**, and the biggest structural change in the edition. It expands 2021's
*Vulnerable and Outdated Components* from "a dependency has a known CVE" into the whole chain:
**build, distribution, update and tooling compromise.** A poisoned package, a compromised CI
runner, a hijacked maintainer account, an unsigned artifact, a typosquatted name, a mutable
action tag.

## Why it is new and ranked third

Because the attack moved. Exploiting an application takes a vulnerability in that
application; compromising its build pipeline reaches every deployment of it at once. The 2021
framing only covered *consuming* a vulnerable component, which missed the direction the
industry was actually being attacked from.

It also ranks high because **it is invisible to code review.** Nothing in reading your own
repository reveals that a transitive dependency changed maintainer last week.

## When to check for it

- **Adding any dependency.** Maintained? Widely used? Reasonably scoped? A tiny convenience
  package is rarely worth a permanent transitive entry.
- **Continuously** — vulnerability scanning, not at release.
- **Whenever the pipeline is touched.** It can deploy code, so it is as privileged as
  production.
- **Before publishing anything others consume.** For this project that is the container image
  and the example policy packs — the one security responsibility self-hosting does not shed.

## How to use it

The standard is `../supply-chain.md`. Pin by digest, scope the pipeline, sign what you ship.

```go
// RIGHT — pinned, with go.sum committed and GOFLAGS=-mod=readonly so a build
// cannot silently alter the graph.
require github.com/open-policy-agent/opa v1.21.2
```

```yaml
# RIGHT — least-privilege pipeline, action pinned by commit, secret scoped to
# the one job that needs it.
permissions:
  contents: read
jobs:
  test:
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2
  release:
    permissions:
      contents: write
    env:
      SIGNING_KEY: ${{ secrets.SIGNING_KEY }}      # not visible to `test`
```

**The invariant: a build is reproducible and an artifact is verifiable.** Pinned versions, a
committed lockfile, integrity hashes, an SBOM, and signed releases.

## How not to use it

**A mutable reference.**

```yaml
# WRONG — v1 is a moving pointer the upstream owner controls.
- uses: some-org/some-action@v1
# Symptom: the action's behaviour changes under you, on a build you did not trigger.
```

**A pipeline that can do anything.**

```yaml
# WRONG
permissions: write-all
# Symptom: one compromised test dependency can publish a release. The pipeline
# becomes the easiest path to production.
```

**Secrets in scope for every job.**

```yaml
# WRONG — the signing key is readable during tests.
env:
  SIGNING_KEY: ${{ secrets.SIGNING_KEY }}
jobs:
  test: ...
  release: ...
# Symptom: a malicious dependency executed by `go test` reads the signing key.
```

**Scanning only at release.** A vulnerability disclosed mid-sprint sits in main until the
release scan finds it — which is the first anyone hears of it.

**Publishing unsigned.** An adopter who cannot verify the artifact cannot distinguish yours
from a replacement.

**Trade-off:** pinning creates a standing obligation to bump deliberately, and for this
project OPA ships monthly minors with tight Go coupling — so the cadence is partly not ours to
choose. The alternative is a dependency graph that changes without anyone deciding.

**Related:** `../supply-chain.md`, `a08-software-or-data-integrity-failures.md`,
`../least-privilege.md`
