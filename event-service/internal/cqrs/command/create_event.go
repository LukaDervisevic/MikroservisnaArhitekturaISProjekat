package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	if c.DateTime <= 0 {
		return status.Error(codes.InvalidArgument, "date_time is required")
	}
	if c.LocationID <= 0 {
		return status.Error(codes.InvalidArgument, "location_id is required")
	}
	return nil
}

type CreateEventHandler struct {
	locationReadRepo repo.ILocationReadRepo
	events           *saga.EventCommands
}

func NewCreateEventHandler(locationReadRepo repo.ILocationReadRepo, events *saga.EventCommands) *CreateEventHandler {
	return &CreateEventHandler{locationReadRepo: locationReadRepo, events: events}
}

func (h *CreateEventHandler) Handle(ctx context.Context, cmd CreateEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}

	id, err := h.events.NextEventID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to allocate event id")
	}

	sagaID, err := h.events.Run(ctx, saga.EventChangeInput{
		EventID: id,
		Op:      saga.OpCreate,
		Payload: saga.NewEventFieldsPayload(saga.EventFields{
			Name: cmd.Name, CotisationPrice: cmd.CotisationPrice, Agenda: cmd.Agenda,
			Type: cmd.Type, DateTime: cmd.DateTime, LocationID: cmd.LocationID,
		}),
	})
	if err != nil {
		log.Error().Err(err).Str("saga_id", sagaID.String()).Int64("event_id", id).
			Msg("create event saga failed and was rolled back")
		return nil, saga.SagaError(sagaID, err)
	}

	return &model.Event{
		Id: id, Name: cmd.Name, CotisationPrice: cmd.CotisationPrice, Agenda: cmd.Agenda,
		Type: cmd.Type, DateTime: cmd.DateTime, LocationID: cmd.LocationID,
	}, nil
}
