package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/service"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/store"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (g *GrpcServer) CreateSourcedEvent(ctx context.Context, req *eventpb.CreateSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	agg, err := g.eventSourcingService.CreateEvent(ctx, service.CreateEventCommand{
		Name: req.Name, CotisationPrice: req.CotisationPrice, Agenda: req.Agenda, Type: req.Type,
		DateTime: req.DateTime.GetSeconds(), LocationID: req.LocationId,
	})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) RenameSourcedEvent(ctx context.Context, req *eventpb.RenameSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.RenameEvent(ctx, service.RenameEventCommand{AggregateID: id, NewName: req.NewName})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) RescheduleSourcedEvent(ctx context.Context, req *eventpb.RescheduleSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.RescheduleEvent(ctx, service.RescheduleEventCommand{AggregateID: id, NewDateTime: req.NewDateTime.GetSeconds()})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) RelocateSourcedEvent(ctx context.Context, req *eventpb.RelocateSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.RelocateEvent(ctx, service.RelocateEventCommand{AggregateID: id, NewLocationID: req.NewLocationId})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) ChangeSourcedEventPrice(ctx context.Context, req *eventpb.ChangeSourcedEventPriceRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.ChangeEventPrice(ctx, service.ChangeEventPriceCommand{AggregateID: id, NewPrice: req.NewPrice})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) CancelSourcedEvent(ctx context.Context, req *eventpb.CancelSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.CancelEvent(ctx, service.CancelEventCommand{AggregateID: id, Reason: req.Reason})
	if err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.SourcedEventMutationResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func (g *GrpcServer) GetSourcedEventState(ctx context.Context, req *eventpb.GetSourcedEventStateRequest) (*eventpb.GetSourcedEventStateResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.Load(ctx, id)
	if err != nil {
		return nil, sourcedEventError(err)
	}
	if !agg.Exists() {
		return nil, status.Error(codes.NotFound, "sourced event not found")
	}
	return &eventpb.GetSourcedEventStateResponse{
		AggregateId:     agg.ID().String(),
		Version:         agg.Version(),
		Name:            agg.Name(),
		CotisationPrice: agg.CotisationPrice(),
		Agenda:          agg.Agenda(),
		Type:            agg.Type(),
		DateTime:        timestampFromUnix(agg.DateTime()),
		LocationId:      agg.LocationID(),
		Cancelled:       agg.Cancelled(),
	}, nil
}

func (g *GrpcServer) GetSourcedEventHistory(ctx context.Context, req *eventpb.GetSourcedEventHistoryRequest) (*eventpb.GetSourcedEventHistoryResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	history, err := g.eventSourcingService.GetHistory(ctx, id)
	if err != nil {
		return nil, sourcedEventError(err)
	}
	pbEvents := make([]*eventpb.SourcedEvent, len(history))
	for i, e := range history {
		pbEvents[i] = &eventpb.SourcedEvent{
			EventId:     e.GetEventID().String(),
			AggregateId: e.GetAggregateID().String(),
			Version:     e.GetVersion(),
			EventType:   e.EventType(),
			OccurredAt:  timestamppb.New(e.GetOccurredAt()),
			Details:     e.Describe(),
		}
	}
	return &eventpb.GetSourcedEventHistoryResponse{Events: pbEvents}, nil
}

func (g *GrpcServer) CreateSourcedEventSnapshot(ctx context.Context, req *eventpb.CreateSourcedEventSnapshotRequest) (*eventpb.CreateSourcedEventSnapshotResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.Load(ctx, id)
	if err != nil {
		return nil, sourcedEventError(err)
	}
	if !agg.Exists() {
		return nil, status.Error(codes.NotFound, "sourced event not found")
	}
	if err := g.eventSourcingService.CreateSnapshot(ctx, agg); err != nil {
		return nil, sourcedEventError(err)
	}
	return &eventpb.CreateSourcedEventSnapshotResponse{AggregateId: agg.ID().String(), Version: agg.Version()}, nil
}

func parseAggregateID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.UUID{}, status.Error(codes.InvalidArgument, "aggregate_id is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, status.Error(codes.InvalidArgument, "aggregate_id must be a valid uuid")
	}
	return id, nil
}

func timestampFromUnix(seconds int64) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(seconds, 0))
}

func sourcedEventError(err error) error {
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
		return status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
}
