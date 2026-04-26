# docker build . -f .\event-service\Dockerfile -t event-service 
FROM golang:1.26.1-alpine AS builder
WORKDIR /event-docker

COPY go.mod go.sum ./
RUN go mod download

COPY event-service/ ./event-service
COPY proto/ ./proto
RUN go build -o main ./event-service/cmd/server
RUN chmod +x ./main

# Runtime
FROM alpine:3.20
WORKDIR /event-docker

COPY --from=builder /event-docker/main .
COPY --from=builder /event-docker/event-service/internal/migrations /event-docker/event-service/internal/migrations

EXPOSE 50055

CMD ["./main"]