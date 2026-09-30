# Repository settings (manual, one time)

These can't be set from files in the repo. Apply them in **Settings**.

## Code security (Settings → Code security)

- [ ] Dependency graph: on
- [ ] Dependabot alerts: on
- [ ] Dependabot security updates: on
- [ ] Secret scanning: on
- [ ] Push protection: on
- [ ] Private vulnerability reporting: on
- [ ] Code scanning: leave off "default setup" (CodeQL runs from `.github/workflows/security.yml`; default setup would conflict with it)

## Branch protection for `main` (Settings → Rules → Rulesets → New branch ruleset)

- Target: default branch
- [ ] Restrict deletions
- [ ] Block force pushes
- [ ] Require a pull request before merging
  - Required approvals: 0 while working solo (raise to 1 when collaborators join)
  - [ ] Require review from Code Owners
  - [ ] Dismiss stale approvals on new commits
- [ ] Require status checks to pass: **CI result** (from `.github/workflows/ci.yml`; it fails if any CI job fails). Also require **Security result** (from `.github/workflows/security.yml`).
- [ ] Require linear history

## General (Settings → General)

- [ ] Allow squash merging only
- [ ] Automatically delete head branches
