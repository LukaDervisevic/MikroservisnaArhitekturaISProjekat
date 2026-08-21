package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

const (
	maxRetries      = 10
	rateLimitPerMin = 10
)

type MailConsumer struct {
	conn      *rmq.AmqpConnection
	consumer  *rmq.Consumer
	publisher *rmq.Publisher
	dlq       *rmq.Publisher
	outbox    *repo.OutboxRepo

	sendQueue string
	dlqQueue  string
}

func NewMailConsumer(
	ctx context.Context,
	conn *rmq.AmqpConnection,
	outbox *repo.OutboxRepo,
	sendQueue string,
	dlqQueue string,
) (*MailConsumer, error) {
	mgmt := conn.Management()
	if _, err := mgmt.DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: sendQueue}); err != nil {
		return nil, err
	}
	if _, err := mgmt.DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: dlqQueue}); err != nil {
		return nil, err
	}

	consumer, err := conn.NewConsumer(ctx, sendQueue, nil)
	if err != nil {
		return nil, err
	}
	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: sendQueue}, nil)
	if err != nil {
		return nil, err
	}
	dlq, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: dlqQueue}, nil)
	if err != nil {
		return nil, err
	}

	return &MailConsumer{
		conn:      conn,
		consumer:  consumer,
		publisher: publisher,
		dlq:       dlq,
		outbox:    outbox,
		sendQueue: sendQueue,
		dlqQueue:  dlqQueue,
	}, nil
}

func (c *MailConsumer) Start(ctx context.Context) {
	interval := time.Minute / time.Duration(rateLimitPerMin)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Msgf("mail consumer started (rate limit %d/min, one message every %s)", rateLimitPerMin, interval)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("mail consumer shutting down")
			return
		case <-ticker.C:
		}

		delivery, err := c.consumer.Receive(ctx)
		if err != nil {
			if ctx.Err() != nil {
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

	err := c.outbox.WriteEmail(email)
	if err == nil {
		log.Info().Str("to", email.To).Int("attempt", email.RetryCount+1).Msg("email written to outbox")
		_ = delivery.Accept(ctx)
		return
	}

	email.RetryCount++
	log.Warn().Err(err).Int("retry_count", email.RetryCount).Str("to", email.To).Msg("email processing failed")

	if email.RetryCount >= maxRetries {
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
