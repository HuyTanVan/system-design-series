FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /review-service ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /review-service /review-service
EXPOSE 8081
ENTRYPOINT ["/review-service"]