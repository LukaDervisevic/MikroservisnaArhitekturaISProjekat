package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := config.LoadEnv(); err != nil {
		panic(err)
	}

	conn := db.Connect()
	if conn == nil {
		panic("Unable to connect to event service database")
	}

	port := os.Getenv("EVENT_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
		return
	}

	eventQueryRepo := repo.NewEventRepo(conn)

	grpcServer := grpc.NewServer()
	event.RegisterEventServiceServer(grpcServer, server.NewGrpcServer(eventQueryRepo))

	go func() {
		log.Printf("starting event query service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	eventQueryQueue := os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE")
	consumerConn, err := rabbitmq.NewConsumerConn(ctx, os.Getenv("RABBITMQ_BROKER_URI"), nil, conn, eventQueryRepo)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to establish consumer connection to rabbitmq broker")
	}
	if err := consumerConn.NewQueueResponder(ctx, eventQueryQueue); err != nil {
		log.Fatal().Err(err).Msgf("unable to create a responder for queue %s", eventQueryQueue)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info().Msg("Shutting down event query gRPC server...")
	grpcServer.GracefulStop()
	_ = consumerConn.Connection.Close(ctx)
	_ = consumerConn.Environment.CloseConnections(ctx)
	log.Info().Msg("Event query gRPC server stopped.")
}
