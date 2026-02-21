# syntax=docker/dockerfile:1

FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/skillz ./cmd/skillz


FROM debian:bookworm-slim

RUN groupadd -r skillz && useradd --no-log-init -r -g skillz skillz

WORKDIR /app

COPY --from=builder /out/skillz /usr/local/bin/skillz

USER skillz

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/skillz"]
