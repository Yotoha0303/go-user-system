# syntax=docker/dockerfile:1

FROM golang:1.25.7-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Install the migration CLI with the same pinned version used by CI.
RUN GOTOOLCHAIN=local go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o go-user-system ./cmd

# === 生产级优化 ===
# 1. 多阶段构建（已完成）
# 2. 非 root 用户
# 3. 最小化镜像 + 安全
# 4. 标签
# 5. 更好的健康检查

FROM alpine:3.22

WORKDIR /app

# 生产安全最佳实践
RUN addgroup -S app && adduser -S app -G app && \
  apk add --no-cache --update ca-certificates mysql-client && \
  chown -R app:app /app

# Copy goose migration tool and SQL migration files
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations/ ./migrations/

COPY --from=builder /app/go-user-system ./go-user-system
COPY config.yml ./config.yml

USER app

EXPOSE 8082

# 生产级健康检查（更严格）
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8082/readyz || exit 1

# 优雅停止
STOPSIGNAL SIGTERM

# 生产推荐标签
LABEL maintainer="go-user-system-v1.0" \
  version="1.0.0" \
  description="go-user-system - Production ready" \
  org.opencontainers.image.source="https://github.com/yotoha/go-user-system"

# 生产推荐使用 ENTRYPOINT + exec form
ENTRYPOINT ["./go-user-system"]
CMD []
