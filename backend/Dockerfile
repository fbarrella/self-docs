# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.27-alpine AS build

WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binaries so the runtime image can be minimal.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- Runtime stage ----
FROM alpine:3.22

# ca-certificates for outbound TLS; wget (busybox) powers the healthcheck.
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/seed /usr/local/bin/seed

USER app
EXPOSE 8080

ENV PORT=8080 GIN_MODE=release

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/server"]
