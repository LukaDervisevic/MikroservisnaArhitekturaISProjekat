package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type MailPublisher struct {
	conn  *PublisherConn
	queue string
}

func NewMailPublisher(ctx context.Context, conn *PublisherConn, queueName string) (*MailPublisher, error) {
	if conn == nil || conn.Connection == nil {
		return nil, fmt.Errorf("nil rabbitmq publisher connection")
	}
	if err := conn.NewQueuePublisher(ctx, conn.Connection, queueName); err != nil {
		return nil, fmt.Errorf("failed to create mail publisher for %s: %w", queueName, err)
	}
	return &MailPublisher{conn: conn, queue: queueName}, nil
}

func (p *MailPublisher) Enqueue(ctx context.Context, to, subject, body string) error {
	return p.PublishEmail(ctx, model.EmailMessage{
		IdempotentKey: uuid.New(),
		To:            to,
		Subject:       subject,
		Body:          body,
		EnqueuedAt:    time.Now().UTC(),
	})
}

func (p *MailPublisher) PublishEmail(ctx context.Context, email model.EmailMessage) error {
	payload, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email message: %w", err)
	}

	if err := p.conn.Publish(ctx, payload, p.queue, true); err != nil {
		log.Error().Err(err).Str("to", email.To).Msg("failed to publish email message")
		return err
	}

	log.Info().
		Str("to", email.To).
		Str("subject", email.Subject).
		Str("id", email.IdempotentKey.String()).
		Str("queue", p.queue).
		Msg("email request enqueued")
	return nil
}
