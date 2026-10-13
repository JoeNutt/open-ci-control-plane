# OWASP Top 10:2025

*Security · External: OWASP Top 10:2025*

One entry per risk category, so a session reads the one it needs. Every entry follows
`../../FORMAT.md`.

## Provenance, and how confident to be

The **2025 edition** was released at Global AppSec in Washington, D.C. in **November 2025** —
the first revision since 2021 — with some sources reporting final confirmation in January
2026. It is built from an analysis of **175,000+ CVEs mapped across 248 CWEs**.

**`owasp.org` could not be read directly** when these entries were written: the Top 10 URL is
a redirect chain the fetcher would not follow. The category identifiers, names and ranking
below are **triangulated across multiple secondary write-ups that agree with each other**, and
they also match the list in the global `sec-threat-modeling` skill independently. Treat the
ranking as reliable and **verify exact official wording against owasp.org before quoting it
in anything published.**

Sources used: [Fastly](https://www.fastly.com/blog/new-2025-owasp-top-10-list-what-changed-what-you-need-to-know) ·
[Patrowl](https://patrowl.io/en/blog/owasp-top-10-2025-what-s-changed-and-the-2026-data) ·
[Invicti](https://www.invicti.com/blog/web-security/owasp-top-10) ·
[Indusface](https://www.indusface.com/learning/owasp-top-10-vulnerabilities/)

## The list

| ID | Category | Movement since 2021 |
|---|---|---|
| `a01-broken-access-control.md` | Broken Access Control | **#1 again.** Now absorbs SSRF; ~40 CWEs, the largest category |
| `a02-security-misconfiguration.md` | Security Misconfiguration | **Up from #5** |
| `a03-software-supply-chain-failures.md` | Software Supply Chain Failures | **New.** Expands 2021's Vulnerable and Outdated Components |
| `a04-cryptographic-failures.md` | Cryptographic Failures | Down from #2 |
| `a05-injection.md` | Injection | Down from #3 |
| `a06-insecure-design.md` | Insecure Design | Down from #4 |
| `a07-authentication-failures.md` | Authentication Failures | Stable. Renamed from Identification and Authentication Failures |
| `a08-software-or-data-integrity-failures.md` | Software or Data Integrity Failures | Stable at #8 |
| `a09-logging-and-alerting-failures.md` | Security Logging and Alerting Failures | Stable at #9. Now emphasises **alerting** |
| `a10-mishandling-of-exceptional-conditions.md` | Mishandling of Exceptional Conditions | **New.** ~24 CWEs |

**Two notes that matter for anyone carrying over 2021 habits.** SSRF is no longer a standalone
category — it is a sub-case of A01, on the reasoning that tricking a server into making a
request it should not is fundamentally an access-control failure. And A10 took the tenth slot
SSRF vacated but is **not** a renamed SSRF; it is a different category about error paths.

## These entries do not restate the standards

The project's own standards live one level up — `../authorization.md`,
`../injection-defense.md`, `../cryptography.md` and the rest. An entry here covers **the risk
category**: what it spans, why it ranks where it does, the question to ask when checking for
it, and the Go shape of getting it wrong. For *how to do it right*, each entry routes to the
standard that owns it.

That split is deliberate: one owner per rule, and no rule stated twice.
