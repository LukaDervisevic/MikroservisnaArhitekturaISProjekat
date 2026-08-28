package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/db"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/grpc/server"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	outboxrepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo/outbox"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/mail"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/outbox"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/ratelimit"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/saga"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
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

	port := os.Getenv("LECTURER_SERVICE_PORT")
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen on specified port")
		return
	}

	lecturerRepo := repo.NewLecturerRepo(conn)
	outboxRepo := outboxrepo.NewOutboxRepo(conn)

	brokerURI := os.Getenv("RABBITMQ_BROKER_URI")
	toLectureQueue := os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE")
	replyToLecturerQueue := os.Getenv("RABBITMQ_REPLY_TO_LECTURER_QUEUE")

	sagaReplies := saga.NewSagaReplyRegistry()

	publisherConn, err := rabbitmq.NewPublisherConn(ctx, brokerURI, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create publisher connection")
	}
	if err := publisherConn.NewQueuePublisher(ctx, publisherConn.Connection, toLectureQueue); err != nil {
		log.Fatal().Err(err).Msgf("failed to create publisher for queue %s", toLectureQueue)
	}

	consumerConn, err := rabbitmq.NewConsumerConn(ctx, brokerURI, nil, conn, sagaReplies)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create consumer connection")
	}
	if err := consumerConn.NewQueueConsumer(ctx, replyToLecturerQueue); err != nil {
		log.Fatal().Err(err).Msgf("failed to start consumer for queue %s", replyToLecturerQueue)
	}

	outboxProcessor := outbox.NewOutboxProcessor(outboxRepo, publisherConn)
	outboxProcessor.StartPoller(ctx, 2*time.Second)

	mailCfg := config.LoadMailConfig()
	mailConsumer, err := startMailWorker(ctx, brokerURI, mailCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start mail worker")
	}
	defer mailConsumer.Close(ctx)

	grpcServer := grpc.NewServer()
	lecturerServer := server.NewGrpcServer(conn, lecturerRepo, outboxRepo, publisherConn, sagaReplies)
	lecturer.RegisterLecturerServiceServer(grpcServer, lecturerServer)

	go func() {
		log.Printf("starting lecturer service grpc server on port %v...", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal().Err(err).Msg("failed to server grpc request")
			cancel()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info().Msg("Shutting down lecturer gRPC server...")
	grpcServer.GracefulStop()

	log.Info().Msg("shutting down rabbitmq lecturer connections...")
	_ = publisherConn.Connection.Close(ctx)
	_ = publisherConn.Environment.CloseConnections(ctx)
	_ = consumerConn.Connection.Close(ctx)
	_ = consumerConn.Environment.CloseConnections(ctx)
	log.Info().Msg("Lecturer gRPC server stopped.")
}

// startMailWorker opens its own broker connection so that the rate-limited mail
// traffic never competes with the saga publishers/consumers.
func startMailWorker(ctx context.Context, brokerURI string, cfg config.MailConfig) (*rabbitmq.MailConsumer, error) {
	env := rmq.NewEnvironment(brokerURI, nil)
	mailConn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail worker: connect to rabbitmq: %w", err)
	}

	if cfg.SMTP.Username == "" || cfg.SMTP.Password == "" {
		log.Warn().
			Str("smtp", cfg.SMTP.Addr()).
			Msg("mail worker: MAIL_SMTP_USERNAME/MAIL_SMTP_PASSWORD are not set, deliveries will be rejected by the relay")
	}

	limiter := ratelimit.NewSlidingWindow(cfg.RateLimit, cfg.RatePeriod)
	worker := mail.NewWorker(limiter, mail.NewSMTPSender(cfg.SMTP))

	consumer, err := rabbitmq.NewMailConsumer(ctx, mailConn, worker, cfg)
	if err != nil {
		return nil, fmt.Errorf("mail worker: create consumer: %w", err)
	}

	go consumer.Start(ctx)
	return consumer, nil
}
