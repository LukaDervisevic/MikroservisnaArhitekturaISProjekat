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
	return nil
}

type UpdateEventHandler struct {
	eventCommandRepo repo.IEventCommandRepo
	locationReadRepo repo.ILocationReadRepo
	orchestrator     *saga.Orchestrator
	definition       saga.Definition
}

func NewUpdateEventHandler(
	eventCommandRepo repo.IEventCommandRepo,
	locationReadRepo repo.ILocationReadRepo,
	orchestrator *saga.Orchestrator,
	definition saga.Definition,
) *UpdateEventHandler {
	return &UpdateEventHandler{
		eventCommandRepo: eventCommandRepo,
		locationReadRepo: locationReadRepo,
		orchestrator:     orchestrator,
		definition:       definition,
	}
}

func (h *UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (*model.Event, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	existing, err := h.eventCommandRepo.GetEventByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if existing == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}

	event := &model.Event{
		Id:              cmd.Id,
		Name:            cmd.Name,
		CotisationPrice: cmd.CotisationPrice,
		Agenda:          cmd.Agenda,
		Type:            cmd.Type,
		DateTime:        cmd.DateTime,
		LocationID:      cmd.LocationID,
	}

	sagaID, err := h.orchestrator.Run(ctx, h.definition, event.Id, saga.UpdateEventInput{
		Event:    event,
		Location: location,
	})
	if err != nil {
		log.Error().Err(err).
			Str("saga_id", sagaID.String()).
			Int64("event_id", event.Id).
			Msg("update event saga did not complete; committed steps were compensated")
		return nil, status.Errorf(codes.Internal, "update event saga %s failed and was rolled back: %v", sagaID, err)
	}

	return event, nil
}
