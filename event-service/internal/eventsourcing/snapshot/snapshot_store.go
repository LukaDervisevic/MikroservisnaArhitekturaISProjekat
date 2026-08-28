package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	"gorm.io/gorm"
)

type SnapshotStore interface {
	Save(ctx context.Context, state aggregate.EventAggregateState) error
	GetLatest(ctx context.Context, aggregateID int64) (*aggregate.EventAggregateState, error)
	WithTx(tx *gorm.DB) SnapshotStore
}

type EventAggregateSnapshot struct {
	AggregateID int64     `gorm:"column:aggregate_id;primaryKey"`
	Version     int64     `gorm:"column:version;primaryKey"`
	State       []byte    `gorm:"column:state"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

type GormSnapshotStore struct {
	db *gorm.DB
}

func NewGormSnapshotStore(db *gorm.DB) *GormSnapshotStore {
	return &GormSnapshotStore{db: db}
}

func (s *GormSnapshotStore) WithTx(tx *gorm.DB) SnapshotStore {
	if tx == nil {
		return s
	}
	return &GormSnapshotStore{db: tx}
}

func (s *GormSnapshotStore) Save(ctx context.Context, state aggregate.EventAggregateState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal snapshot state: %w", err)
	}
	record := EventAggregateSnapshot{
		AggregateID: state.ID,
		Version:     state.Version,
		State:       payload,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func (s *GormSnapshotStore) GetLatest(ctx context.Context, aggregateID int64) (*aggregate.EventAggregateState, error) {
	var record EventAggregateSnapshot
	err := s.db.WithContext(ctx).
		Where("aggregate_id = ?", aggregateID).
		Order("version DESC").
		Limit(1).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load latest snapshot: %w", err)
	}

	var state aggregate.EventAggregateState
	if err := json.Unmarshal(record.State, &state); err != nil {
		return nil, fmt.Errorf("decode snapshot state: %w", err)
	}
	return &state, nil
}
