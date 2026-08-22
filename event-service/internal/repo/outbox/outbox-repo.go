package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type OutboxRepo struct {
	db *gorm.DB
}

const (
	StatusStashed = iota
	StatusSent
)

func NewOutboxRepo(db *gorm.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) WithTx(tx *gorm.DB) *OutboxRepo {
	if tx == nil {
		return r
	}
	return &OutboxRepo{db: tx}
}

func (r *OutboxRepo) StashMessage(ctx context.Context, message rabbitmq.Message) error {
	encPayload, err := json.Marshal(message)
	if err != nil {
		log.Error().Msgf("message %s not encoded into payload", message.IdempotentKey.String())
		return err
	}

	outbox := &model.Outbox{
		ID:        uuid.New(),
		Status:    StatusStashed,
		Timestamp: time.Now(),
		Payload:   encPayload,
	}
	return r.db.WithContext(ctx).Create(outbox).Error
}

func (r *OutboxRepo) GetStashedMessages(ctx context.Context, limit int) ([]model.Outbox, error) {
	var outboxMessages []model.Outbox
	err := r.db.WithContext(ctx).
		Where("status = ?", StatusStashed).
		Order("timestamp ASC").
		Limit(limit).
		Find(&outboxMessages).Error

	return outboxMessages, err
}

func (r *OutboxRepo) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Outbox{}).
		Where("id = ?", id).
		Update("status", StatusSent).Error
}
