package service

import (
	"context"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/snapshot"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const snapshotEvery = 5

type EventAggregateService struct {
	store     store.EventStore
	snapshots snapshot.SnapshotStore
}

func NewEventAggregateService(eventStore store.EventStore, snapshotStore snapshot.SnapshotStore) *EventAggregateService {
	return &EventAggregateService{store: eventStore, snapshots: snapshotStore}
}

func (s *EventAggregateService) Load(ctx context.Context, id uuid.UUID) (*aggregate.EventAggregate, error) {
	snap, err := s.snapshots.GetLatest(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	var agg *aggregate.EventAggregate
	fromVersion := int64(1)
	if snap != nil {
		agg = aggregate.FromState(*snap)
		fromVersion = snap.Version + 1
	} else {
		agg = aggregate.NewEventAggregate(id)
	}

	tail, err := s.store.LoadFrom(ctx, id, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("load event tail: %w", err)
	}
	agg.ReplayHistory(tail)
	return agg, nil
}

func (s *EventAggregateService) save(ctx context.Context, agg *aggregate.EventAggregate) error {
	uncommitted := agg.UncommittedEvents()
	if len(uncommitted) == 0 {
		return nil
	}
	if err := s.store.Append(ctx, uncommitted); err != nil {
		return err
	}
	agg.MarkCommitted()

	if agg.Version()%snapshotEvery == 0 {
		if err := s.CreateSnapshot(ctx, agg); err != nil {
			return fmt.Errorf("auto snapshot at version %d: %w", agg.Version(), err)
		}
	}
	return nil
}

func (s *EventAggregateService) CreateSnapshot(ctx context.Context, agg *aggregate.EventAggregate) error {
	return s.snapshots.Save(ctx, agg.ToState())
}

func (s *EventAggregateService) GetHistory(ctx context.Context, id uuid.UUID) ([]domainevent.DomainEvent, error) {
	return s.store.Load(ctx, id)
}

type CreateEventCommand struct {
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

func (c CreateEventCommand) Validate() error {
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if c.Agenda == "" {
		return status.Error(codes.InvalidArgument, "agenda is required")
	}
	if c.Type == "" {
		return status.Error(codes.InvalidArgument, "type is required")
	}
	if c.LocationID <= 0 {
		return status.Error(codes.InvalidArgument, "location_id is required")
	}
	return nil
}

func (s *EventAggregateService) CreateEvent(ctx context.Context, cmd CreateEventCommand) (*aggregate.EventAggregate, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	agg := aggregate.NewEventAggregate(uuid.New())
	if err := agg.Create(cmd.Name, cmd.CotisationPrice, cmd.Agenda, cmd.Type, cmd.DateTime, cmd.LocationID); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

type RenameEventCommand struct {
	AggregateID uuid.UUID
	NewName     string
}

func (s *EventAggregateService) RenameEvent(ctx context.Context, cmd RenameEventCommand) (*aggregate.EventAggregate, error) {
	agg, err := s.Load(ctx, cmd.AggregateID)
	if err != nil {
		return nil, err
	}
	if err := agg.Rename(cmd.NewName); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

type RescheduleEventCommand struct {
	AggregateID uuid.UUID
	NewDateTime int64
}

func (s *EventAggregateService) RescheduleEvent(ctx context.Context, cmd RescheduleEventCommand) (*aggregate.EventAggregate, error) {
	agg, err := s.Load(ctx, cmd.AggregateID)
	if err != nil {
		return nil, err
	}
	if err := agg.Reschedule(cmd.NewDateTime); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

type RelocateEventCommand struct {
	AggregateID   uuid.UUID
	NewLocationID int64
}

func (s *EventAggregateService) RelocateEvent(ctx context.Context, cmd RelocateEventCommand) (*aggregate.EventAggregate, error) {
	agg, err := s.Load(ctx, cmd.AggregateID)
	if err != nil {
		return nil, err
	}
	if err := agg.Relocate(cmd.NewLocationID); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

type ChangeEventPriceCommand struct {
	AggregateID uuid.UUID
	NewPrice    float64
}

func (s *EventAggregateService) ChangeEventPrice(ctx context.Context, cmd ChangeEventPriceCommand) (*aggregate.EventAggregate, error) {
	agg, err := s.Load(ctx, cmd.AggregateID)
	if err != nil {
		return nil, err
	}
	if err := agg.ChangePrice(cmd.NewPrice); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}

type CancelEventCommand struct {
	AggregateID uuid.UUID
	Reason      string
}

func (s *EventAggregateService) CancelEvent(ctx context.Context, cmd CancelEventCommand) (*aggregate.EventAggregate, error) {
	agg, err := s.Load(ctx, cmd.AggregateID)
	if err != nil {
		return nil, err
	}
	if err := agg.Cancel(cmd.Reason); err != nil {
		return nil, err
	}
	if err := s.save(ctx, agg); err != nil {
		return nil, err
	}
	return agg, nil
}
