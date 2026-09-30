# Security findings: triage process

Every PR runs `.github/workflows/security.yml`. Its **Security result** check
is required on `main`, so a real finding blocks the merge until someone
triages it. Results also land in **Security → Code scanning**, grouped by
tool (the SARIF "category").

## What runs, and what fails the build

| Area | Tool | Scope | Fails the build on |
| --- | --- | --- | --- |
| SAST | CodeQL (`security-extended`) | Go, TypeScript | Any alert at error level (via code scanning's PR check) |
| SAST | gosec | `backend/` | Any finding |
| SCA | govulncheck | Go module + stdlib | Any vulnerability in reachable code |
| SCA | npm audit | `frontend/` | Advisories rated high or critical |
| SCA | Trivy filesystem | lockfiles, `go.mod` | Fixable HIGH or CRITICAL |
| Secrets | gitleaks | full git history | Any leak |
| IaC | Checkov | Dockerfiles, workflows | Any failed check |
| IaC | KICS | `docker-compose.yml` | MEDIUM and above |
| Container | Trivy image | both images | Fixable HIGH or CRITICAL vulns, or embedded secrets |

Lower severities are uploaded to the Security tab but do not block. The
weekly scheduled run catches CVEs published after the code last changed.

## Triage: pick exactly one outcome

1. **Fix.** The default. Upgrade the dependency or base image, change the
   code, or tighten the config. Dependabot usually has the upgrade ready.
2. **Accept the risk.** Only when a fix isn't available or isn't worth it
   *and* the finding isn't exploitable here. Needs a written justification
   and a review date (at most 90 days out).
3. **False positive.** The tool is wrong about this code. Needs a one-line
   reason that a reviewer can check.

Accepted risks and false positives are suppressed **in the repo, next to
the thing they suppress, in the same PR that introduces them**, so the
suppression itself gets code review (CODEOWNERS routes `.github/` to the
owner). Never dismiss an alert only in the Security tab UI: that leaves no
review trail and the next scan re-raises it.

## How to suppress, per tool

Every suppression carries a reason. A bare suppression fails review.

| Tool | Where | Format |
| --- | --- | --- |
| gosec | the line | `// #nosec G404 -- IDs here are non-security; see #NN` |
| CodeQL | Security tab, after the PR merges | Dismiss as "false positive" or "won't fix" *with a comment linking the PR or issue that justified it* |
| govulncheck | not suppressible | Fix, or document why the vulnerable path is unreachable in an issue |
| npm audit | `frontend/package.json` `overrides` | Force a patched transitive version; note why in the PR |
| Trivy | `.trivyignore` at the repo root | `CVE-2026-12345 exp:2026-12-31 # not reachable: we never call X` |
| gitleaks | `.gitleaksignore` | the finding's fingerprint, with a comment. A real secret is never ignored: **rotate it first**, then remove it from history if needed |
| Checkov | the resource | `# checkov:skip=CKV_DOCKER_2: healthcheck defined in compose` |
| KICS | the compose file | `# kics-scan ignore-line` with a reason on the line above |

Expiring suppressions (`exp:` in `.trivyignore`) re-open themselves, which
forces the review date to happen.

## When a gate fails on a PR

1. Open the failing job; each scanner prints its findings in the log.
2. Find the same alert in **Security → Code scanning** (filter by the PR
   branch) for the full description and remediation.
3. Choose an outcome above. For a fix, push to the same PR. For a
   suppression, add it with its reason in the same PR and call it out in
   the PR description under *Security checklist*.
4. If the failure is in a dependency the PR didn't touch (a new CVE in an
   existing package), fix it in a separate PR so the original change stays
   reviewable, and link the two.

## Known gaps

- Checkov silently skips a workflow its schema rejects (for example an
  expression in `concurrency.cancel-in-progress`). If a workflow change makes
  Checkov's `github_actions` pass count drop, check for this.
- `npm audit` has no SARIF output; its advisories reach the Security tab
  through the Trivy filesystem scan instead.
- Private vulnerability reporting (see `SECURITY.md`) is the intake for
  findings from outside the pipeline; triage them the same way.
