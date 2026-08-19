FROM golang:1.23.0-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bin/api ./cmd/api

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
RUN apk add --no-cache ca-certificates wget
USER app

WORKDIR /app
COPY --from=builder /app/bin/api /app/api

EXPOSE 8080

ENTRYPOINT ["/app/api"]

