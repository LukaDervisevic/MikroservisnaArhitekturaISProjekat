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
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/service/saga"
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

	brokerURI := os.Getenv("RABBITMQ_BROKER_URI")

	fromLecturerQueue := os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE")
	fromEventQueue := os.Getenv("RABBITMQ_EVENT_TO_LECTURE_QUEUE")
	replyToLectureQueue := os.Getenv("RABBITMQ_REPLY_TO_LECTURE_QUEUE")
	toLectureQueryQueue := os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE")
	replyToLecturerQueue := os.Getenv("RABBITMQ_REPLY_TO_LECTURER_QUEUE")

	mailQueue := os.Getenv("RABBITMQ_MAIL_QUEUE")
	mailDLQQueue := os.Getenv("RABBITMQ_MAIL_DLQ_QUEUE")
	outboxDir := os.Getenv("MAIL_OUTBOX_DIR")

	publisherConn, err := rabbitmq.NewPublisherConn(ctx, brokerURI, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create publisher connection")
	}
	for _, queue := range []string{toLectureQueryQueue, replyToLecturerQueue} {
		if err := publisherConn.NewQueueRequester(ctx, publisherConn.Connection, queue); err != nil {
			log.Fatal().Err(err).Msgf("failed to create requester for queue %s", queue)
		}
	}

	sagaReplies := saga.NewSagaReplyRegistry()
	lectureRepo := repo.NewLectureRepo(conn)
	eventRepo := repo.NewEventRepo(conn)
	lecturerRepo := repo.NewLecturerRepo(conn)

	consumerConn, err := rabbitmq.NewConsumerConn(ctx, brokerURI, nil, conn, eventRepo, lecturerRepo, lectureRepo, sagaReplies, publisherConn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create consumer connection")
	}

	for _, queue := range []string{fromLecturerQueue, fromEventQueue, replyToLectureQueue} {
		if err := consumerConn.NewQueueResponder(ctx, queue); err != nil {
			log.Fatal().Err(err).Msgf("failed to start responder for queue %s", queue)
		}
	}

	port := os.Getenv("LECTURE_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
	}

	grpcServer := grpc.NewServer()
	lecture.RegisterLectureServiceServer(grpcServer, server.NewGrpcServer(conn, publisherConn, sagaReplies, lectureRepo, eventRepo, lecturerRepo))

	go func() {
		log.Printf("starting lecture service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to serve grpc request")
			cancel()
		}
	}()

	go func() {
		env := rmq.NewEnvironment(brokerURI, nil)
		mailConn, err := env.NewConnection(ctx)
		if err != nil {
			log.Error().Err(err).Msg("mail worker: failed to connect to rabbitmq")
			return
		}

		outbox, err := repo.NewOutboxRepo(outboxDir)
		if err != nil {
			log.Error().Err(err).Msg("mail worker: failed to init outbox")
			return
		}

		consumer, err := rabbitmq.NewMailConsumer(ctx, mailConn, outbox, mailQueue, mailDLQQueue)
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

	log.Info().Msg("Shutting down lecture gRPC server...")
	grpcServer.GracefulStop()
	_ = consumerConn.Connection.Close(ctx)
	_ = consumerConn.Environment.CloseConnections(ctx)
	log.Info().Msg("Lecture gRPC server stopped.")
}
