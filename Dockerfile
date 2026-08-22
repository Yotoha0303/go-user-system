# syntax=docker/dockerfile:1

FROM golang:1.25.13-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -trimpath \
  -ldflags="-s -w -X go-user-system/internal/buildinfo.Version=${VERSION} -X go-user-system/internal/buildinfo.Commit=${COMMIT} -X go-user-system/internal/buildinfo.BuildTime=${BUILD_TIME}" \
  -o go-user-system ./cmd

FROM alpine:3.22

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app && \
  apk add --no-cache ca-certificates && \
  chown -R app:app /app

COPY --chown=app:app migrations/ ./migrations/
COPY --chown=app:app --from=builder /app/go-user-system ./go-user-system
COPY --chown=app:app config.yml ./config.yml

USER app

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8082/readyz || exit 1

STOPSIGNAL SIGTERM

LABEL maintainer="go-user-system-v1.0" \
  description="Go user authentication and RBAC backend" \
  org.opencontainers.image.source="https://github.com/Yotoha0303/go-user-system" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${COMMIT}" \
  org.opencontainers.image.created="${BUILD_TIME}"

ENTRYPOINT ["/app/go-user-system"]
CMD []
