package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
)

type MailPublisher struct {
	publisher *rmq.Publisher
}

func NewMailPublisher(ctx context.Context, conn *rmq.AmqpConnection, queueName string) (*MailPublisher, error) {
	if conn == nil {
		return nil, fmt.Errorf("nil rabbitmq connection")
	}

	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: queueName}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher for %s: %w", queueName, err)
	}

	return &MailPublisher{publisher: publisher}, nil
}

func (p *MailPublisher) PublishEmail(ctx context.Context, email model.EmailMessage) error {
	body, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email message: %w", err)
	}

	msg := rmq.NewMessage(body)
	if _, err := p.publisher.Publish(ctx, msg); err != nil {
		log.Error().Err(err).Str("to", email.To).Msg("failed to publish email message")
		return err
	}

	log.Info().Str("to", email.To).Str("id", email.IdempotentKey.String()).Msg("email message published to queue")
	return nil
}

func (p *MailPublisher) Close(ctx context.Context) {
	if p.publisher != nil {
		_ = p.publisher.Close(ctx)
	}
}
