package outbox

import (
	"context"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo/outbox"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type OutboxProcessor struct {
	db            *gorm.DB
	outboxRepo    *outbox.OutboxRepo
	publisherConn *rabbitmq.PublisherConn
}

func NewOutboxProcessor(db *gorm.DB, outboxRepo *outbox.OutboxRepo, brokerConn *rabbitmq.PublisherConn) *OutboxProcessor {
	return &OutboxProcessor{
		db:            db,
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

// TODO: Add offset when i enable data streaming
func (p *OutboxProcessor) ProcessOutbox(ctx context.Context) error {
	var stashed []model.Outbox

	err := p.db.WithContext(ctx).
		Where("status = ?", outbox.StatusStashed).
		Order("timestamp ASC").
		Limit(10).
		Find(&stashed).Error

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

		err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Model(&model.Outbox{}).
				Where("id = ?", record.ID).
				Update("status", outbox.StatusSent).Error
		})

		if err != nil {
			log.Error().Err(err).Msgf("failed to mark outbox record %s as sent", record.ID)
		} else {
			log.Info().Msgf("successfully published and marked outbox message %s", record.ID)
		}
	}

	return nil
}
