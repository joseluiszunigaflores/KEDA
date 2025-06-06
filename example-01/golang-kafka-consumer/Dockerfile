FROM golang:1.23-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/kafka-consumer .
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/kafka-consumer .
CMD ["./kafka-consumer"]