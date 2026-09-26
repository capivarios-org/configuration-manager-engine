# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/data-plane ./cmd/server

# Final Stage (Scratch/Distroless ultra-leve)
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/data-plane /app/data-plane

EXPOSE 8080

ENTRYPOINT ["/app/data-plane"]
