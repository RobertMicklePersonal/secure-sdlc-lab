# syntax=docker/dockerfile:1

# Base images are pinned by digest (Dependabot keeps them current). The tag
# is kept alongside for readability.

# ---- build ----
FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
WORKDIR /src

# Download modules first so this layer caches across source changes.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
# Static, stripped, reproducible binary: no cgo, no local paths, no VCS stamp.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags='-s -w' -o /out/api ./cmd/api

# ---- runtime ----
# distroless/static has no shell, package manager or libc: just CA certs,
# tzdata and /etc/passwd. The :nonroot variant runs as UID 65532.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3

LABEL org.opencontainers.image.source="https://github.com/RobertMicklePersonal/secure-sdlc-lab" \
      org.opencontainers.image.description="secure-sdlc-lab notes API"

COPY --from=build /out/api /api

USER 65532:65532
EXPOSE 8080
ENV ADDR=:8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/api", "-healthcheck"]

ENTRYPOINT ["/api"]
