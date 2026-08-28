package saga

import (
	"context"
	"errors"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	esservice "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/service"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventCommands struct {
	orchestrator *Orchestrator
	definition   Definition
	service      *esservice.EventAggregateService
}

func NewEventCommands(orchestrator *Orchestrator, definition Definition, service *esservice.EventAggregateService) *EventCommands {
	return &EventCommands{orchestrator: orchestrator, definition: definition, service: service}
}

func (c *EventCommands) NextEventID(ctx context.Context) (int64, error) {
	return c.service.NextAggregateID(ctx)
}

func (c *EventCommands) Load(ctx context.Context, id int64) (*aggregate.EventAggregate, error) {
	return c.service.Load(ctx, id)
}

func (c *EventCommands) Run(ctx context.Context, in EventChangeInput) (uuid.UUID, error) {
	return c.orchestrator.Run(ctx, c.definition, in.EventID, in)
}

func SagaError(sagaID uuid.UUID, err error) error {
	if s, ok := status.FromError(err); ok && s.Code() != codes.Unknown {
		return err
	}
	switch {
	case errors.Is(err, aggregate.ErrDoesNotExist):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, aggregate.ErrAlreadyExists),
		errors.Is(err, aggregate.ErrEventCancelled),
		errors.Is(err, aggregate.ErrAlreadyCancelled),
		errors.Is(err, aggregate.ErrNameRequired),
		errors.Is(err, aggregate.ErrAgendaRequired),
		errors.Is(err, aggregate.ErrTypeRequired),
		errors.Is(err, aggregate.ErrInvalidDateTime),
		errors.Is(err, aggregate.ErrInvalidLocation),
		errors.Is(err, aggregate.ErrInvalidPrice),
		errors.Is(err, aggregate.ErrNoOpChange):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, store.ErrConcurrentModification):
		return status.Error(codes.Aborted, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("event saga %s failed and was rolled back: %v", sagaID, err))
	}
}
