package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	aggregateTypeEvent = "Event"
	eventIDSequence    = "event_service.events_id_seq"
)

type EventStoreEvent struct {
	EventID       uuid.UUID `gorm:"column:event_id;primaryKey;type:uuid"`
	AggregateID   int64     `gorm:"column:aggregate_id"`
	AggregateType string    `gorm:"column:aggregate_type"`
	Version       int64     `gorm:"column:version"`
	EventType     string    `gorm:"column:event_type"`
	Payload       []byte    `gorm:"column:payload"`
	OccurredAt    time.Time `gorm:"column:occurred_at"`
}

type GormEventStore struct {
	db *gorm.DB
}

func NewGormEventStore(db *gorm.DB) *GormEventStore {
	return &GormEventStore{db: db}
}

func (s *GormEventStore) WithTx(tx *gorm.DB) EventStore {
	if tx == nil {
		return s
	}
	return &GormEventStore{db: tx}
}

func (s *GormEventStore) NextAggregateID(ctx context.Context) (int64, error) {
	var id int64
	if err := s.db.WithContext(ctx).
		Raw("SELECT nextval(?)", eventIDSequence).
		Scan(&id).Error; err != nil {
		return 0, fmt.Errorf("mint aggregate id: %w", err)
	}
	return id, nil
}

func (s *GormEventStore) Delete(ctx context.Context, aggregateID int64) error {
	return s.db.WithContext(ctx).
		Where("aggregate_id = ?", aggregateID).
		Delete(&EventStoreEvent{}).Error
}

func (s *GormEventStore) Append(ctx context.Context, events []domainevent.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	records := make([]EventStoreEvent, 0, len(events))
	for _, e := range events {
		payload, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", e.EventType(), err)
		}
		records = append(records, EventStoreEvent{
			EventID:       e.GetEventID(),
			AggregateID:   e.GetAggregateID(),
			AggregateType: aggregateTypeEvent,
			Version:       e.GetVersion(),
			EventType:     e.EventType(),
			Payload:       payload,
			OccurredAt:    e.GetOccurredAt(),
		})
	}

	if err := s.db.WithContext(ctx).Create(&records).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrConcurrentModification
		}
		return fmt.Errorf("append events: %w", err)
	}
	return nil
}

func (s *GormEventStore) Load(ctx context.Context, aggregateID int64) ([]domainevent.DomainEvent, error) {
	return s.LoadFrom(ctx, aggregateID, 1)
}

func (s *GormEventStore) LoadFrom(ctx context.Context, aggregateID int64, fromVersion int64) ([]domainevent.DomainEvent, error) {
	var records []EventStoreEvent
	err := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND version >= ?", aggregateID, fromVersion).
		Order("version ASC").
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	events := make([]domainevent.DomainEvent, 0, len(records))
	for _, r := range records {
		event, err := decodeEvent(r.EventType, r.Payload)
		if err != nil {
			return nil, fmt.Errorf("decode event %s (version %d): %w", r.EventID, r.Version, err)
		}
		events = append(events, event)
	}
	return events, nil
}

func decodeEvent(eventType string, payload []byte) (domainevent.DomainEvent, error) {
	var event domainevent.DomainEvent
	switch eventType {
	case domainevent.TypeEventCreated:
		event = &domainevent.EventCreated{}
	case domainevent.TypeEventRenamed:
		event = &domainevent.EventRenamed{}
	case domainevent.TypeEventRescheduled:
		event = &domainevent.EventRescheduled{}
	case domainevent.TypeEventRelocated:
		event = &domainevent.EventRelocated{}
	case domainevent.TypeEventPriceChanged:
		event = &domainevent.EventPriceChanged{}
	case domainevent.TypeEventAgendaChanged:
		event = &domainevent.EventAgendaChanged{}
	case domainevent.TypeEventTypeChanged:
		event = &domainevent.EventTypeChanged{}
	case domainevent.TypeEventCancelled:
		event = &domainevent.EventCancelled{}
	case domainevent.TypeEventUncancelled:
		event = &domainevent.EventUncancelled{}
	default:
		return nil, fmt.Errorf("unknown event type %q", eventType)
	}
	if err := json.Unmarshal(payload, event); err != nil {
		return nil, err
	}
	return event, nil
}

const pgUniqueViolation = "23505"

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgUniqueViolation
	}
	return false
}
