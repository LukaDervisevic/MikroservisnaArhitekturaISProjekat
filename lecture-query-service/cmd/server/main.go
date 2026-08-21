package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
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
		panic("Unable to connect to lecture query service database")
	}

	port := os.Getenv("LECTURER_QUERY_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
		return
	}

	grpcServer := grpc.NewServer()
	lecture.RegisterLectureServiceServer(grpcServer, server.NewGrpcServer(conn))

	go func() {
		log.Printf("starting lecturer query service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	var lecturerConsumerConn rabbitmq.ConsumerConn

	go func(brokerURI string, queue string) {
		lectureQueryRepo := repo.NewLectureQueryRepo(conn)
		broker := rabbitmq.NewConsumerConn(ctx, brokerURI, nil, conn, lectureQueryRepo)

		err := broker.NewQueueResponder(ctx, queue)
		if err != nil {
			return
		}
	}(
		os.Getenv("RABBITMQ_BROKER_URI"),
		os.Getenv("RABBITMQ_LECTURE_QUERY_QUEUE"),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	defer func() { _ = lecturerConsumerConn.Environment.CloseConnections(ctx) }()
	defer func() { _ = lecturerConsumerConn.Connection.Close(ctx) }()

	log.Info().Msg("Shutting down lectuer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
