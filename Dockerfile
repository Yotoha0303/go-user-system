# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o go-user-system ./cmd

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /app/go-user-system ./go-user-system
COPY config.yml ./config.yml

USER app

EXPOSE 8082

CMD ["./go-user-system"]
