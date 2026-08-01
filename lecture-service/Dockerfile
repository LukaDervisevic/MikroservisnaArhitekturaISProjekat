FROM golang:1.26.1-alpine AS builder
WORKDIR /lecture-docker

COPY go.mod go.sum ./
RUN go mod download

COPY lecture-service/ ./lecture-service
COPY proto/ ./proto
RUN go build -o main ./lecture-service/cmd/server
RUN chmod +x ./main

# Runtime
FROM alpine:3.20
WORKDIR /lecture-docker

COPY --from=builder /lecture-docker/main .
COPY --from=builder /lecture-docker/lecture-service/internal/migrations /lecture-docker/internal/migrations
COPY --from=builder /lecture-docker/lecture-service/.env.docker /lecture-docker/lecture-service/.env.docker

EXPOSE 50055

CMD ["./main"]