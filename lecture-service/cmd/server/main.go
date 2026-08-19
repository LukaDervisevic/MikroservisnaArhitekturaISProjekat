package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
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
		panic("Unable to connect to lecture service database")
	}

	port := os.Getenv("LECTURE_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
		return
	}

	grpcServer := grpc.NewServer()
	lecture.RegisterLectureServiceServer(grpcServer, server.NewGrpcServer(ctx, conn))

	go func() {
		log.Printf("starting lecture service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	var lecturerConsumerConn rabbitmq.ConsumerConn

	go func(brokerURI string, queue string) {
		eventRepo := repo.NewEventRepo(conn)
		lecturerRepo := repo.NewLecturerRepo(conn)
		broker, _ := rabbitmq.NewBrokerServerConn(ctx, brokerURI, nil, conn, eventRepo, lecturerRepo)

		if err := broker.NewQueueResponder(ctx, "lecture-events"); err != nil {
			return
		}
		if err := broker.NewQueueResponder(ctx, "lecturers"); err != nil {
			return
		}

	}(
		os.Getenv("RABBITMQ_BROKER_URI"),
		os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE"),
	)

	go func() {
		brokerURI := os.Getenv("RABBITMQ_BROKER_URI")

		env := rmq.NewEnvironment(brokerURI, nil)
		mailConn, err := env.NewConnection(ctx)
		if err != nil {
			log.Error().Err(err).Msg("mail worker: failed to connect to rabbitmq")
			return
		}

		outbox, err := repo.NewOutboxRepo("outbox")
		if err != nil {
			log.Error().Err(err).Msg("mail worker: failed to init outbox")
			return
		}

		consumer, err := rabbitmq.NewMailConsumer(ctx, mailConn, outbox, "mail.send", "mail.dlq")
		if err != nil {
			log.Error().Err(err).Msg("mail worker: failed to create consumer")
			return
		}
		defer consumer.Close(ctx)

		consumer.Start(ctx)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	defer func() { _ = lecturerConsumerConn.Environment.CloseConnections(ctx) }()
	defer func() { _ = lecturerConsumerConn.Connection.Close(ctx) }()

	log.Info().Msg("Shutting down lectuer gRPC server...")
	grpcServer.GracefulStop()
	log.Info().Msg("Lecturer gRPC server stopped.")

}
