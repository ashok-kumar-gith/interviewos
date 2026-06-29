# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.22-alpine AS builder

# git is needed for module resolution of some VCS-hosted deps.
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads.
COPY go.mod go.sum* ./
RUN go mod download

# Build the API binary (static, stripped).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- Runtime stage ----
# Distroless static: tiny, non-root, no shell.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# CA certs for outbound TLS (Claude API, OAuth providers).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/api /app/api

# Ship the OpenAPI spec alongside the binary for the /swagger endpoint.
COPY --from=builder /src/api/openapi.yaml /app/api/openapi.yaml

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/api"]
