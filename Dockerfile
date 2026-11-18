# ---------- Stage 1: build ----------
# go.mod menggunakan direktif go 1.26.0 (mengikuti dependency validator v10,
# x/crypto, x/net, x/text terbaru), jadi builder memakai Go 1.26.
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependency download terlebih dahulu.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/monetry-api .

# ---------- Stage 2: runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S monetry \
    && adduser -S -G monetry monetry

WORKDIR /app

COPY --from=builder /out/monetry-api /app/monetry-api

# Migration di-embed ke binary, tetapi folder tetap disalin agar
# mudah diinspeksi saat demo.
COPY --from=builder /src/migrations /app/migrations

USER monetry

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/monetry-api"]
