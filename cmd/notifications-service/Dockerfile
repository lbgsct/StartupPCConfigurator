FROM golang:1.23.3-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o notifications ./cmd/notifications-service

FROM alpine:3.17
WORKDIR /app

COPY --from=builder /app/notifications /app/notifications

EXPOSE 8004
CMD ["/app/notifications"]
