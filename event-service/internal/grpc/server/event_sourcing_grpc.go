package server

import (
	"context"
	"strconv"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (g *GrpcServer) mutateSourcedEvent(ctx context.Context, in saga.EventChangeInput) (*eventpb.SourcedEventMutationResponse, error) {
	sagaID, err := g.events.Run(ctx, in)
	if err != nil {
		return nil, saga.SagaError(sagaID, err)
	}
	agg, err := g.events.Load(ctx, in.EventID)
	if err != nil {
		return nil, status.Error(codes.Internal, "loaded event after mutation failed")
	}
	return &eventpb.SourcedEventMutationResponse{
		AggregateId: strconv.FormatInt(agg.ID(), 10),
		Version:     agg.Version(),
	}, nil
}

func (g *GrpcServer) CreateSourcedEvent(ctx context.Context, req *eventpb.CreateSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	id, err := g.events.NextEventID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to allocate event id")
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id,
		Op:      saga.OpCreate,
		Payload: saga.NewEventFieldsPayload(saga.EventFields{
			Name: req.Name, CotisationPrice: req.CotisationPrice, Agenda: req.Agenda,
			Type: req.Type, DateTime: req.DateTime.GetSeconds(), LocationID: req.LocationId,
		}),
	})
}

func (g *GrpcServer) RenameSourcedEvent(ctx context.Context, req *eventpb.RenameSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id, Op: saga.OpRename, Payload: saga.NewNamePayload(req.NewName),
	})
}

func (g *GrpcServer) RescheduleSourcedEvent(ctx context.Context, req *eventpb.RescheduleSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id, Op: saga.OpReschedule, Payload: saga.NewDateTimePayload(req.NewDateTime.GetSeconds()),
	})
}

func (g *GrpcServer) RelocateSourcedEvent(ctx context.Context, req *eventpb.RelocateSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id, Op: saga.OpRelocate, Payload: saga.NewLocationPayload(req.NewLocationId),
	})
}

func (g *GrpcServer) ChangeSourcedEventPrice(ctx context.Context, req *eventpb.ChangeSourcedEventPriceRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id, Op: saga.OpChangePrice, Payload: saga.NewPricePayload(req.NewPrice),
	})
}

func (g *GrpcServer) CancelSourcedEvent(ctx context.Context, req *eventpb.CancelSourcedEventRequest) (*eventpb.SourcedEventMutationResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	return g.mutateSourcedEvent(ctx, saga.EventChangeInput{
		EventID: id, Op: saga.OpCancel, Payload: saga.NewReasonPayload(req.Reason),
	})
}

func (g *GrpcServer) GetSourcedEventState(ctx context.Context, req *eventpb.GetSourcedEventStateRequest) (*eventpb.GetSourcedEventStateResponse, error) {
	id, err := parseAggregateID(req.GetAggregateId())
	if err != nil {
		return nil, err
	}
	agg, err := g.eventSourcingService.Load(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !agg.Exists() {
		return nil, status.Error(codes.NotFound, "sourced event not found")
	}
	return &eventpb.GetSourcedEventStateResponse{
		AggregateId:     strconv.FormatInt(agg.ID(), 10),
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbEvents := make([]*eventpb.SourcedEvent, len(history))
	for i, e := range history {
		pbEvents[i] = &eventpb.SourcedEvent{
			EventId:     e.GetEventID().String(),
			AggregateId: strconv.FormatInt(e.GetAggregateID(), 10),
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
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !agg.Exists() {
		return nil, status.Error(codes.NotFound, "sourced event not found")
	}
	if err := g.eventSourcingService.CreateSnapshot(ctx, agg); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &eventpb.CreateSourcedEventSnapshotResponse{
		AggregateId: strconv.FormatInt(agg.ID(), 10),
		Version:     agg.Version(),
	}, nil
}

func parseAggregateID(raw string) (int64, error) {
	if raw == "" {
		return 0, status.Error(codes.InvalidArgument, "aggregate_id is required")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, status.Error(codes.InvalidArgument, "aggregate_id must be a positive integer")
	}
	return id, nil
}

func timestampFromUnix(seconds int64) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(seconds, 0))
}
