# Supply chain

*Security · Derived from the global `sec-review-testing` skill — **binding on this project***

## What it is

Dependencies and the build pipeline treated as attack surface. **Your dependencies run with
your privileges**, and your pipeline can deploy code, so it is at least as privileged as
production. Keep a transitive inventory, pin and verify, vet before adding, and give the
pipeline its own least-privilege review.

## Why it exists

Because it is now among the most exploited risk categories, and it **postdates most
secure-coding material** — so it is the control most often missing from an otherwise careful
threat model. Nothing in a code review of your own repository reveals that a transitive
dependency changed maintainer last week.

For this project specifically it is the one security responsibility **self-hosting does not
shed**: the published container image and the example policy packs reach every adopter.

## When to use it

- **Adding any dependency.** Is it maintained, widely used, reasonably scoped? A tiny
  convenience package is rarely worth a new supply-chain entry.
- **Continuously** — scan for known vulnerabilities, not at release.
- **Whenever the pipeline is touched.** Secrets in CI need the same care as secrets in
  production.
- **Before publishing an artifact** others consume: signed releases, an SBOM, verifiable
  builds, a documented disclosure process.

Watch for typosquats and sudden maintainer changes.

## How to use it

```go
// RIGHT — pinned with an integrity hash. go.sum is committed, and
// GOFLAGS=-mod=readonly means a build cannot silently change the graph.
require (
    github.com/open-policy-agent/opa v1.21.2
)
```

```yaml
# RIGHT — the pipeline runs as least privilege, with pinned actions and no
# inherited write scope.
permissions:
  contents: read            # not write-all
jobs:
  test:
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683   # v4.2.2, pinned by SHA
```

```makefile
# RIGHT — the checks that must run before an artifact is published.
release: sbom sign verify
	@echo "artifact signed, SBOM attached, hashes verified"
```

**The invariant: the build pipeline is production infrastructure.** If it can deploy, it is as
privileged as the thing it deploys, and it needs the same scrutiny.

## How not to use it

**A floating version.**

```go
// WRONG — "latest" means the build is not reproducible and a compromise of the
// upstream account reaches you on the next build.
require github.com/some/dep v0.0.0-00010101000000-000000000000   // replaced, unpinned
```

```yaml
# WRONG — a mutable tag. The action can be changed under you.
- uses: some-org/some-action@v1
# Symptom: v1 is a moving pointer. Pin the commit SHA.
```

**A pipeline with write-all permissions.**

```yaml
# WRONG — every job can push code, tags and packages.
permissions: write-all
# Symptom: one compromised test dependency can publish a release. The pipeline
# is now the easiest path to production.
```

**Secrets available to every job.**

```yaml
# WRONG — the signing key is in scope for the test job too.
env:
  SIGNING_KEY: ${{ secrets.SIGNING_KEY }}
jobs:
  test: ...
  release: ...
# Symptom: a malicious dependency executed during tests can read the signing key.
# Scope it to the job that signs, in a separate privileged job.
```

**Scanning at release only.** A vulnerability disclosed mid-sprint is in your main branch for
the rest of the sprint, and the release scan is the first anyone hears of it.

**Adding a dependency for one function.** Each one is a permanent transitive entry, a
maintainer you are trusting, and an update you now owe.

**Trade-off:** pinning means a standing obligation to bump deliberately, which is real,
recurring work — and for this project OPA ships monthly minors with tight Go coupling, so the
cadence is partly not ours to choose. The alternative is a dependency graph that changes
without anyone deciding.

**Related:** `least-privilege.md`, `secret-management.md`, `security-review-and-testing.md`
