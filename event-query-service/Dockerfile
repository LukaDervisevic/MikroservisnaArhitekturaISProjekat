FROM golang:1.26.1-alpine AS builder
WORKDIR /event-query-docker

COPY go.mod go.sum ./
RUN go mod download

COPY event-query-service/ ./event-query-service
COPY proto/ ./proto
RUN go build -o main ./event-query-service/cmd/server
RUN chmod +x ./main

# Runtime
FROM alpine:3.20
WORKDIR /event-query-docker

COPY --from=builder /event-query-docker/main .
COPY --from=builder /event-query-docker/event-query-service/internal/migrations /event-query-docker/internal/migrations

EXPOSE 50055

CMD ["./main"]