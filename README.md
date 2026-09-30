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

## Layout

```
backend/    Go API (standard library net/http, in-memory store)
frontend/   React + TypeScript + Vite UI
.github/    workflows, templates, Dependabot
docs/       setup notes and runbooks
```

## Local development

Needs Go 1.27+, Node 22+ and make. Docker is optional, for `make up`.

```sh
make install   # npm ci in frontend/
make dev       # API on :8080, UI on http://localhost:5173
make test      # go test -race + vitest
make lint      # gofmt, go vet, golangci-lint, oxlint, tsc
```

To run the hardened containers instead:

```sh
make up        # docker compose: UI + API on http://localhost:8080
make down
```

In dev, Vite proxies `/api` to the Go server, so the browser sees one origin and CORS stays closed.

### API

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/healthz` | Liveness |
| GET | `/api/notes` | List, newest first |
| POST | `/api/notes` | `{"title", "body"}`, returns 201 |
| GET | `/api/notes/{id}` | |
| PUT | `/api/notes/{id}` | Replace title and body |
| DELETE | `/api/notes/{id}` | Returns 204 |

Configuration is by environment variable:

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADDR` | `:8080` | Listen address |
| `ALLOWED_ORIGINS` | empty (no CORS) | Comma-separated exact origins allowed to call the API from a browser |
| `MAX_NOTES` | `1000` | Store capacity, bounds memory use |

### Security controls in the app

- **Input:** JSON only (415 otherwise), 64 KiB body cap (413), unknown fields and trailing data rejected, title 1-200 and body 0-5000 characters, no control characters (422).
- **IDs:** 128-bit random, format-checked before lookup, so they can't be guessed or used for path tricks.
- **Server:** read-header, read, write and idle timeouts plus a 16 KiB header cap against slow-client DoS; graceful shutdown.
- **Headers:** `Content-Security-Policy: default-src 'none'`, `nosniff`, `X-Frame-Options: DENY`, `no-referrer`, `no-store`.
- **CORS:** exact-match allowlist, never reflects arbitrary origins, never allows credentials.
- **Logging:** one JSON line per request with a request ID; no headers or note contents are logged, and panics return a generic 500.
- **Containers:** multi-stage builds; the API runs from `distroless/static` (no shell or package manager) as UID 65532, the UI from `nginx-unprivileged` as UID 101; base images pinned by digest. Compose runs both with a read-only root filesystem, all capabilities dropped, `no-new-privileges`, PID, memory and CPU limits, and healthchecks. Only the UI port is published, on 127.0.0.1; the API is internal.
- **UI headers:** nginx serves a strict CSP (`default-src 'self'`, no inline scripts or styles, `frame-ancestors 'none'`), plus `nosniff`, `DENY`, `no-referrer` and a restrictive `Permissions-Policy`; `server_tokens off` and a 64 KiB body cap.
- **UI:** React escapes all note text (no `dangerouslySetInnerHTML`); inputs mirror server limits.

## CI

`.github/workflows/ci.yml` runs on every pull request and on pushes to `main`: Go (gofmt, vet, golangci-lint, `go test -race` with coverage), frontend (`npm ci --ignore-scripts`, oxlint, tsc, vitest, build) and a build of both Docker images. The `CI result` job summarizes them and is the check the `main` ruleset requires.

The workflow is hardened as supply-chain surface: `permissions: contents: read`, every action pinned to a commit SHA (Dependabot bumps them), `persist-credentials: false`, no `pull_request_target`, and stale PR runs cancelled.

## Reporting a vulnerability

See [SECURITY.md](SECURITY.md).
