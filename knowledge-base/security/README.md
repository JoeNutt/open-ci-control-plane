# Secure coding

Every entry here is **derived from a global `sec-*` skill and binding on this project**. The
skill remains the authority; if an entry disagrees with its skill, the skill wins and the
entry is the bug.

**Security is a design property.** Most serious vulnerabilities are decisions, not typos —
and a decision is far cheaper to change before it is implemented. Start at
`threat-modeling.md`.

| Entry | Read it for |
|---|---|
| `threat-modeling.md` | The principles that decide most questions, trust boundaries, STRIDE |
| `input-validation.md` | Allowlists, canonicalise-then-validate, Unicode, resource bounds |
| `injection-defense.md` | Structural separation of code and data, per-interpreter; SSRF, XXE, deserialisation |
| `authorization.md` | Deny by default, and authorising the **object** rather than only the action |
| `least-privilege.md` | Minimum rights for minimum time, for processes, tokens and roles |
| `cryptography.md` | Choosing primitives, and the misuses that break correct maths |
| `secret-management.md` | Where keys live, rotation, and what must never be logged |
| `error-handling-and-logging.md` | Failing closed, two audiences, and logging for detection |
| `supply-chain.md` | Dependencies and the build pipeline as attack surface |
| `security-review-and-testing.md` | Reviewing by exposure, testing the denial, fuzzing parsers |
| `owasp-top-10/` | The **OWASP Top 10:2025** risk categories, one entry each. Read after a design exists, as a checklist against it. |

## Proportionality

Match the effort to what is actually at risk. **Say so plainly when a security measure is not
warranted** — recommending controls nobody needs trains people to ignore the recommendations
that matter.
