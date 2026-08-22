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

	lectureQueryRepo := repo.NewLectureQueryRepo(conn)

	grpcServer := grpc.NewServer()
	lecture.RegisterLectureServiceServer(grpcServer, server.NewGrpcServer(lectureQueryRepo))

	go func() {
		log.Printf("starting lecturer query service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	brokerURI := os.Getenv("RABBITMQ_BROKER_URI")
	lectureQueryQueue := os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE")
	replyToLectureQueue := os.Getenv("RABBITMQ_REPLY_TO_LECTURE_QUEUE")

	publisherConn, err := rabbitmq.NewPublisherConn(ctx, brokerURI, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create publisher connection")
	}
	if err := publisherConn.NewQueueRequester(ctx, publisherConn.Connection, replyToLectureQueue); err != nil {
		log.Fatal().Err(err).Msgf("failed to create requester for queue %s", replyToLectureQueue)
	}

	consumerConn, err := rabbitmq.NewConsumerConn(ctx, brokerURI, nil, conn, lectureQueryRepo, publisherConn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create consumer connection")
	}
	if err := consumerConn.NewQueueResponder(ctx, lectureQueryQueue); err != nil {
		log.Fatal().Err(err).Msgf("failed to start responder for queue %s", lectureQueryQueue)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	defer func() { _ = consumerConn.Environment.CloseConnections(ctx) }()
	defer func() { _ = consumerConn.Connection.Close(ctx) }()

	log.Info().Msg("Shutting down lectuer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
