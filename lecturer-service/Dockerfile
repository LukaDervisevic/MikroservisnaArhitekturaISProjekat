FROM golang:1.26.1-alpine AS builder
WORKDIR /lecturer-docker

COPY go.mod go.sum ./
RUN go mod download

COPY lecturer-service/ ./lecturer-service
COPY proto/ ./proto
RUN go build -o main ./lecturer-service/cmd/server
RUN chmod +x ./main

FROM alpine:3.20
WORKDIR /lecturer-docker

COPY  --from=builder /lecturer-docker/main . 
COPY --from=builder /lecturer-docker/lecturer-service/internal/migrations /lecturer-docker/lecturer-service/internal/migrations


EXPOSE 50054

CMD ["./main"]  