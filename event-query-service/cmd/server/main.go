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

	grpcServer := grpc.NewServer()
	event.RegisterEventServiceServer(grpcServer, server.NewGrpcServer(conn))

	go func() {
		log.Printf("starting event query service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	var serverConn rabbitmq.ConsumerConn

	go func(brokerURI string, queue string) {
		eventQueryRepo := repo.NewEventRepo(conn)
		serverConn, err := rabbitmq.NewConsumerConn(ctx, brokerURI, nil, conn, eventQueryRepo)
		if err != nil {
			log.Error().Msg("unable to establish consumer connection to rabbitmq broker")
			return
		}
		err = serverConn.NewQueueResponder(ctx, queue)
		if err != nil {
			log.Error().Msg("unable to create a rabbitmq queue responder")
			return
		}

	}(
		os.Getenv("RABBITMQ_BROKER_URI"),
		os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE"),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	defer func() { _ = serverConn.Environment.CloseConnections(ctx) }()
	defer func() { _ = serverConn.Connection.Close(ctx) }()

	log.Info().Msg("Shutting down lectuer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
