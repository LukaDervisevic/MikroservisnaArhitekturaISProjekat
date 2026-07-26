package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
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

func (r *OutboxRepo) StashMessage(ctx context.Context, message rabbitmq.Message[interface{}]) error {
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
