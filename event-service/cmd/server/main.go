package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/location"
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

	eventRepo := repo.NewEventRepo(conn)
	locationRepo := repo.NewLocationRepo(conn)

	queryQueue := os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE")
	queryBroker, err := rabbitmq.NewPublisherConn(ctx, os.Getenv("RABBITMQ_BROKER_URI"), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create publisher connection")
	}
	if err := queryBroker.NewQueueRequester(ctx, queryBroker.Connection, queryQueue); err != nil {
		log.Fatal().Err(err).Msgf("failed to create requester for queue %s", queryQueue)
	}

	eventServer := server.NewGrpcServer(conn, eventRepo, locationRepo, queryBroker)

	grpcServer := grpc.NewServer()
	event.RegisterEventServiceServer(grpcServer, eventServer)
	location.RegisterLocationServiceServer(grpcServer, eventServer)

	go func() {
		log.Printf("starting event service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info().Msg("Shutting down event gRPC server...")
	grpcServer.GracefulStop()
	_ = queryBroker.Connection.Close(ctx)
	_ = queryBroker.Environment.CloseConnections(ctx)
	log.Info().Msg("Event gRPC server stopped.")
}
