FROM golang:1.23-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go mod tidy
RUN go build -o /app/kafka-producer .
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/kafka-producer .
CMD ["./kafka-producer"]