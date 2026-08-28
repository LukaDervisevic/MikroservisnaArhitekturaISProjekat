FROM golang:1.26.1-alpine AS builder
WORKDIR /lecture-query-docker

COPY go.mod go.sum ./
RUN go mod download

COPY lecture-query-service/ ./lecture-query-service
COPY proto/ ./proto
RUN go build -o main ./lecture-query-service/cmd/server
RUN chmod +x ./main

# Runtime
FROM alpine:3.20
WORKDIR /lecture-query-docker

COPY --from=builder /lecture-query-docker/main .
COPY --from=builder /lecture-query-docker/lecture-query-service/internal/migrations /lecture-query-docker/internal/migrations

EXPOSE 50055

CMD ["./main"]