package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
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

	port := os.Getenv("LECTURER_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
		return
	}

	grpcServer := grpc.NewServer()
	lecturerServer := server.NewGrpcServer(ctx, conn)
	lecturer.RegisterLecturerServiceServer(grpcServer, lecturerServer)

	go func() {
		log.Printf("starting lecturer service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	/*
		go func(brokerURI string, queue string, replyQueue string) {
			log.Info().Msgf("connection attempt to RabbitMQ message broker at %s", brokerURI)
			clientConn = rabbitmq.NewRabbitMQClientConn(ctx, brokerURI, nil)
			clientConn.NewQueueRequester(ctx, clientConn.Connection, queue)
		}(
			os.Getenv("RABBITMQ_BROKER_URI"),
			os.Getenv("RABBITMQ_LECTURER_QUEUE"),
			os.Getenv("RABBITMQ_LECTURER_REPLY_QUEUE"),
		)
	*/
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info().Msgf("shutting down rabbitmq lecturer client...")
	defer func() { _ = lecturerServer.BrokerConn.Environment.CloseConnections(ctx) }()
	defer func() { _ = lecturerServer.BrokerConn.Requester.Close(ctx) }()
	defer func() { _ = lecturerServer.BrokerConn.Connection.Close(ctx) }()
	log.Info().Msgf("successfully shut down rabbitmq connection")

	log.Info().Msg("Shutting down lecturer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
