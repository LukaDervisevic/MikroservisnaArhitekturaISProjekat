package client

import (
	"io"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/grpc/interceptors"
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewLecturerClient(addr string, serviceName string, cb *interceptors.CircuitBreaker) (lecturerpb.LecturerServiceClient, io.Closer, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			interceptors.CircuitBreakerInterceptor(serviceName, cb),
			interceptors.RetryInterceptor,
			interceptors.TimeoutInterceptor,
			interceptors.LoggerInterceptor))
	if err != nil {
		return nil, nil, err
	}
	return lecturerpb.NewLecturerServiceClient(conn), conn, nil
}
