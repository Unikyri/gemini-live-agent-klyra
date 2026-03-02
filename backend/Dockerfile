# Build stage — compiled with the official Go image
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o klyra-backend ./cmd/api/main.go

# Runtime stage — minimal scratch image for security and size
FROM gcr.io/distroless/static-debian12 AS runner

WORKDIR /app
COPY --from=builder /app/klyra-backend .

# The port must match the PORT env var (default: 8080)
EXPOSE 8080

ENTRYPOINT ["/app/klyra-backend"]
