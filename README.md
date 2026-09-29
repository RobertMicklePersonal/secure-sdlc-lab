# secure-sdlc-lab

A hands-on lab for practicing a full **secure software development lifecycle** on a small
Go API + React (TypeScript, Vite) web app.

## What this repo exercises

| Area | Tooling |
| --- | --- |
| Source control & planning | GitHub, Issues, PR templates, branch protection, CODEOWNERS |
| CI | GitHub Actions: lint, test, build on every PR |
| SAST | CodeQL (Go + TypeScript), gosec |
| SCA | govulncheck, npm audit, Trivy filesystem scan, Dependabot |
| Secrets | GitHub secret scanning + push protection |
| IaC | Checkov (Dockerfiles, compose, workflows) |
| Containers | Multi-stage, distroless, non-root images; Trivy image scan |
| CD | Build, sign (cosign), SBOM, publish to GitHub Container Registry |
| DAST & QA | OWASP ZAP baseline scan, Playwright end-to-end tests |

## Roadmap

Work is tracked as GitHub Issues, one per phase:

1. Repo foundation (#1)
2. Application (Go API + React UI) with unit tests (#2)
3. Dockerize (#3)
4. CI pipeline (#4)
5. Security gates (SAST, SCA, secrets, IaC, image) (#5)
6. CD: sign and publish to GHCR (#6)
7. DAST and end-to-end QA (#7)

## Layout (planned)

```
backend/    Go API
frontend/   React + TypeScript + Vite UI
.github/    workflows, templates, Dependabot
docs/       setup notes and runbooks
```

## Reporting a vulnerability

See [SECURITY.md](SECURITY.md).
