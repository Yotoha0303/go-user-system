# syntax=docker/dockerfile:1

FROM node:22.22.2-alpine3.23 AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.28-alpine

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/ || exit 1

LABEL org.opencontainers.image.source="https://github.com/Yotoha0303/go-user-system" \
  org.opencontainers.image.description="Go User System web client" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${COMMIT}" \
  org.opencontainers.image.created="${BUILD_TIME}"
