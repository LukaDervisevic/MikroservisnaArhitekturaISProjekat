package outbox

import (
	"context"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo/outbox"
	"github.com/rs/zerolog/log"
)

type OutboxProcessor struct {
	outboxRepo    *outbox.OutboxRepo
	publisherConn *rabbitmq.PublisherConn
}

func NewOutboxProcessor(outboxRepo *outbox.OutboxRepo, brokerConn *rabbitmq.PublisherConn) *OutboxProcessor {
	return &OutboxProcessor{
		outboxRepo:    outboxRepo,
		publisherConn: brokerConn,
	}
}

func (p *OutboxProcessor) StartPoller(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := p.ProcessOutbox(ctx); err != nil {
					log.Error().Err(err).Msg("error processing outbox messages")
				}
			case <-ctx.Done():
				ticker.Stop()
				log.Info().Msg("outbox poller stopped")
				return
			}
		}
	}()
}

const outboxBatchSize = 10

func (p *OutboxProcessor) ProcessOutbox(ctx context.Context) error {
	stashed, err := p.outboxRepo.GetStashedMessages(ctx, outboxBatchSize)
	if err != nil || len(stashed) == 0 {
		return err
	}

	for _, record := range stashed {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := p.publisherConn.Publish(pubCtx, record.Payload, os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE"), true)
		cancel()

		if err != nil {
			log.Error().Err(err).Msgf("failed to publish outbox record %s, skipping status update", record.ID)
			continue
		}

		if err := p.outboxRepo.MarkAsSent(ctx, record.ID); err != nil {
			log.Error().Err(err).Msgf("failed to mark outbox record %s as sent", record.ID)
		} else {
			log.Info().Msgf("successfully published and marked outbox message %s", record.ID)
		}
	}

	return nil
}
