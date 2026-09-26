FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /search-service ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /search-service /search-service
EXPOSE 8083
ENTRYPOINT ["/search-service"]