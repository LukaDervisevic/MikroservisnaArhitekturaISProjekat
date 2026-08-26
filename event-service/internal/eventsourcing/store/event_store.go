package store

import (
	"context"
	"errors"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"github.com/google/uuid"
)

var ErrConcurrentModification = errors.New("concurrent modification: aggregate was changed by another writer")

type EventStore interface {
	Append(ctx context.Context, events []domainevent.DomainEvent) error
	Load(ctx context.Context, aggregateID uuid.UUID) ([]domainevent.DomainEvent, error)
	LoadFrom(ctx context.Context, aggregateID uuid.UUID, fromVersion int64) ([]domainevent.DomainEvent, error)
}
