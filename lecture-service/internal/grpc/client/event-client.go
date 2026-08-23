package client

import (
	"io"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/grpc/interceptors"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewEventClient(addr string) (eventpb.EventServiceClient, io.Closer, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			interceptors.RetryInterceptor,
			interceptors.TimeoutInterceptor,
			interceptors.LoggerInterceptor))
	if err != nil {
		return nil, nil, err
	}
	return eventpb.NewEventServiceClient(conn), conn, nil
}
