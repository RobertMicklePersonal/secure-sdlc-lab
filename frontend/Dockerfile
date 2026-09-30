# syntax=docker/dockerfile:1

# Base images are pinned by digest (Dependabot keeps them current). The tag
# is kept alongside for readability.

# ---- build ----
FROM node:26-alpine@sha256:0b36e8c136b94cd4fcf02188228e76c31ad5872eef3fec8cbd2eee500cfd9e80 AS build
WORKDIR /src

# Install from the lockfile only, without running package install scripts.
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund

COPY . .
RUN npm run build

# ---- runtime ----
# nginx-unprivileged runs as UID 101 and listens on 8080, so it needs no
# root and no NET_BIND_SERVICE capability.
FROM nginxinc/nginx-unprivileged:1.30-alpine@sha256:ed04ec1ff34502c339ee5c3ae3f855442398edc1d05591e2b98981dcbbd20b1e

LABEL org.opencontainers.image.source="https://github.com/RobertMicklePersonal/secure-sdlc-lab" \
      org.opencontainers.image.description="secure-sdlc-lab notes UI"

COPY nginx/default.conf nginx/security-headers.inc /etc/nginx/conf.d/
COPY --from=build /src/dist /usr/share/nginx/html

USER 101:101
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/healthz"]
