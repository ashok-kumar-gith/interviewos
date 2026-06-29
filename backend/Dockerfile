# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.24-alpine AS builder

# git is needed for module resolution of some VCS-hosted deps.
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads.
COPY go.mod go.sum* ./
RUN go mod download

# Build all three binaries (static, stripped): the API server plus the migrate
# and seed commands (the latter two are invoked by the K8s/Helm migrate+seed Jobs).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api     ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/seed    ./cmd/seed

# ---- Runtime stage ----
# Distroless static: tiny, non-root, no shell.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# CA certs for outbound TLS (Claude API, OAuth providers).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/api /app/api
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /out/seed /app/seed

# Ship the OpenAPI spec alongside the binary for the /swagger endpoint.
COPY --from=builder /src/api/openapi.yaml /app/api/openapi.yaml

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/api"]
