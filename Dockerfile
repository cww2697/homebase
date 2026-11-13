# Multi-stage Dockerfile for Homebase Dashboard
# Stage 1: Build static Go executable
FROM golang:alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /homebase main.go

# Stage 2: Minimal runtime image with Git CLI and SSL certificates
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata git openssh-client

WORKDIR /app

# Copy binary from builder
COPY --from=builder /homebase /app/homebase

# Create default data directory for persistent configuration
RUN mkdir -p /app/data
VOLUME ["/app/data"]

ENV PORT=8080
ENV HOMEBASE_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/homebase"]
CMD ["-data=/app/data", "-port=8080"]
