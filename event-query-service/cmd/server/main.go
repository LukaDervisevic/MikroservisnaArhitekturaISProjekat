package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {

	_, cancel := context.WithCancel(context.Background())
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
		log.Printf("starting event service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	//var serverConn rabbitmq.BrokerServerConn[model.EventWithLocation]
	//
	//go func(brokerURI string, queue string) {
	//	log.Info().Msgf("connection attempt to RabbitMQ message broker at %s", brokerURI)
	//	lecturerRepo := repo.NewLecturerRepo(conn)
	//	serverConn := rabbitmq.NewBrokerServerConn[model.EventWithLocation](ctx, brokerURI, nil, conn, lecturerRepo)
	//
	//	serverConn.NewQueueResponder(ctx, serverConn.Connection, queue)
	//
	//}(
	//	os.Getenv("RABBITMQ_BROKER_URI"),
	//	os.Getenv("RABBITMQ_LECTURER_QUEUE"),
	//)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	//defer func() { _ = serverConn.Environment.CloseConnections(ctx) }()
	//defer func() { _ = serverConn.Responder.Close(ctx) }()
	//defer func() { _ = serverConn.Connection.Close(ctx) }()

	log.Info().Msg("Shutting down lectuer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
