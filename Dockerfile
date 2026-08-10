# syntax=docker/dockerfile:1

FROM golang:1.25.12-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Install the migration CLI with the same pinned version used by CI.
RUN GOTOOLCHAIN=local go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o go-user-system ./cmd

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app && \
  apk add --no-cache ca-certificates && \
  chown -R app:app /app

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --chown=app:app migrations/ ./migrations/
COPY --chown=app:app --from=builder /app/go-user-system ./go-user-system
COPY --chown=app:app config.yml ./config.yml

USER app

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8082/readyz || exit 1

STOPSIGNAL SIGTERM

LABEL maintainer="go-user-system-v1.0" \
  version="1.0.0-rc.3" \
  description="Go user authentication and RBAC backend" \
  org.opencontainers.image.source="https://github.com/Yotoha0303/go-user-system"

ENTRYPOINT ["/app/go-user-system"]
CMD []
