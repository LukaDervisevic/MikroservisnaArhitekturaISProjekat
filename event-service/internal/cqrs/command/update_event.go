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

type UpdateEventCommand struct {
	Id              int64
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

func (c UpdateEventCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
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

type UpdateEventHandler struct {
	locationReadRepo repo.ILocationReadRepo
	events           *saga.EventCommands
}

func NewUpdateEventHandler(locationReadRepo repo.ILocationReadRepo, events *saga.EventCommands) *UpdateEventHandler {
	return &UpdateEventHandler{locationReadRepo: locationReadRepo, events: events}
}

func (h *UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	current, err := h.events.Load(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load event")
	}
	if !current.Exists() || current.Cancelled() {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	result := &model.Event{
		Id: cmd.Id, Name: cmd.Name, CotisationPrice: cmd.CotisationPrice, Agenda: cmd.Agenda,
		Type: cmd.Type, DateTime: cmd.DateTime, LocationID: cmd.LocationID,
	}

	if current.Name() == cmd.Name && current.CotisationPrice() == cmd.CotisationPrice &&
		current.Agenda() == cmd.Agenda && current.Type() == cmd.Type &&
		current.DateTime() == cmd.DateTime && current.LocationID() == cmd.LocationID {
		return result, nil
	}

	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}

	sagaID, err := h.events.Run(ctx, saga.EventChangeInput{
		EventID: cmd.Id,
		Op:      saga.OpUpdate,
		Payload: saga.NewEventFieldsPayload(saga.EventFields{
			Name: cmd.Name, CotisationPrice: cmd.CotisationPrice, Agenda: cmd.Agenda,
			Type: cmd.Type, DateTime: cmd.DateTime, LocationID: cmd.LocationID,
		}),
	})
	if err != nil {
		log.Error().Err(err).Str("saga_id", sagaID.String()).Int64("event_id", cmd.Id).
			Msg("update event saga failed and was rolled back")
		return nil, saga.SagaError(sagaID, err)
	}

	return result, nil
}
