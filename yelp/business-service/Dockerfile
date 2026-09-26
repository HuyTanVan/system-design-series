FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /business-service ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /business-service /business-service
EXPOSE 8081
ENTRYPOINT ["/business-service"]