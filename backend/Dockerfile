# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/indieforge ./cmd/indieforge

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/indieforge ./indieforge

EXPOSE 8080
ENTRYPOINT ["./indieforge"]
