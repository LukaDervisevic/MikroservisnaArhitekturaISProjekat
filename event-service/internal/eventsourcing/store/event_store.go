package store

import (
	"context"
	"errors"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"gorm.io/gorm"
)

var ErrConcurrentModification = errors.New("concurrent modification: aggregate was changed by another writer")

type EventStore interface {
	Append(ctx context.Context, events []domainevent.DomainEvent) error
	Load(ctx context.Context, aggregateID int64) ([]domainevent.DomainEvent, error)
	LoadFrom(ctx context.Context, aggregateID int64, fromVersion int64) ([]domainevent.DomainEvent, error)
	Delete(ctx context.Context, aggregateID int64) error
	NextAggregateID(ctx context.Context) (int64, error)
	WithTx(tx *gorm.DB) EventStore
}
