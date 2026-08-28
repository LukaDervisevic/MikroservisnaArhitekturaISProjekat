package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/config"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/mail"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type MailConsumer struct {
	consumer  *rmq.Consumer
	publisher *rmq.Publisher
	dlq       *rmq.Publisher
	worker    *mail.Worker
	cfg       config.MailConfig
}

func NewMailConsumer(
	ctx context.Context,
	conn *rmq.AmqpConnection,
	worker *mail.Worker,
	cfg config.MailConfig,
) (*MailConsumer, error) {
	mgmt := conn.Management()
	if _, err := mgmt.DeclareQueue(ctx, &rmq.ClassicQueueSpecification{Name: cfg.Queue}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return nil, err
		}
	}
	if _, err := mgmt.DeclareQueue(ctx, &rmq.ClassicQueueSpecification{Name: cfg.DLQQueue}); err != nil {
		if !errors.Is(err, rmq.ErrPreconditionFailed) {
			return nil, err
		}
	}

	consumer, err := conn.NewConsumer(ctx, cfg.Queue, nil)
	if err != nil {
		return nil, err
	}
	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: cfg.Queue}, nil)
	if err != nil {
		return nil, err
	}
	dlq, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: cfg.DLQQueue}, nil)
	if err != nil {
		return nil, err
	}

	return &MailConsumer{
		consumer:  consumer,
		publisher: publisher,
		dlq:       dlq,
		worker:    worker,
		cfg:       cfg,
	}, nil
}

func (c *MailConsumer) Start(ctx context.Context) {
	log.Info().
		Int("limit", c.cfg.RateLimit).
		Dur("period", c.cfg.RatePeriod).
		Str("queue", c.cfg.Queue).
		Str("smtp", c.cfg.SMTP.Addr()).
		Msg("mail consumer started")

	for {
		if ctx.Err() != nil {
			log.Info().Msg("mail consumer shutting down")
			return
		}

		delivery, err := c.consumer.Receive(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info().Msg("mail consumer shutting down")
				return
			}
			log.Error().Err(err).Msg("failed to receive message")
			continue
		}

		c.handle(ctx, delivery)
	}
}

func (c *MailConsumer) handle(ctx context.Context, delivery rmq.IDeliveryContext) {
	msg := delivery.Message()

	var payload []byte
	if len(msg.Data) > 0 {
		payload = msg.Data[0]
	}

	var email model.EmailMessage
	if err := json.Unmarshal(payload, &email); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal email message, discarding")
		_ = delivery.Accept(ctx)
		return
	}

	err := c.worker.Process(ctx, email)
	if err == nil {
		_ = delivery.Accept(ctx)
		return
	}
	if ctx.Err() != nil {
		_ = delivery.Requeue(ctx)
		return
	}

	email.RetryCount++
	log.Warn().Err(err).Int("retry_count", email.RetryCount).Str("to", email.To).Msg("email processing failed")

	if email.RetryCount >= c.cfg.MaxRetries {
		if pubErr := c.republish(ctx, c.dlq, email); pubErr != nil {
			log.Error().Err(pubErr).Msg("failed to publish to DLQ, requeueing original")
			_ = delivery.Requeue(ctx)
			return
		}
		log.Error().Str("to", email.To).Int("retry_count", email.RetryCount).Msg("email moved to DLQ after max retries")
		_ = delivery.Accept(ctx)
		return
	}

	if pubErr := c.republish(ctx, c.publisher, email); pubErr != nil {
		log.Error().Err(pubErr).Msg("failed to re-publish for retry, requeueing original")
		_ = delivery.Requeue(ctx)
		return
	}
	_ = delivery.Accept(ctx)
}

func (c *MailConsumer) republish(ctx context.Context, pub *rmq.Publisher, email model.EmailMessage) error {
	body, err := json.Marshal(email)
	if err != nil {
		return err
	}
	_, err = pub.Publish(ctx, rmq.NewMessage(body))
	return err
}

func (c *MailConsumer) Close(ctx context.Context) {
	if c.consumer != nil {
		_ = c.consumer.Close(ctx)
	}
	if c.publisher != nil {
		_ = c.publisher.Close(ctx)
	}
	if c.dlq != nil {
		_ = c.dlq.Close(ctx)
	}
}
