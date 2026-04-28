FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.mod ./
RUN go mod download
COPY . .
RUN go build -o batch ./cmd/batch

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/batch .
CMD ["./batch"]