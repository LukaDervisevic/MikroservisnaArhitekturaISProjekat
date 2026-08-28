package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteEventCommand struct {
	Id int64
}

func (c DeleteEventCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteEventHandler struct {
	events *saga.EventCommands
}

func NewDeleteEventHandler(events *saga.EventCommands) *DeleteEventHandler {
	return &DeleteEventHandler{events: events}
}

func (h *DeleteEventHandler) Handle(ctx context.Context, cmd DeleteEventCommand) (*model.Event, error) {
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

	deleted := &model.Event{
		Id: current.ID(), Name: current.Name(), CotisationPrice: current.CotisationPrice(),
		Agenda: current.Agenda(), Type: current.Type(), DateTime: current.DateTime(),
		LocationID: current.LocationID(),
	}

	sagaID, err := h.events.Run(ctx, saga.EventChangeInput{
		EventID: cmd.Id,
		Op:      saga.OpCancel,
		Payload: saga.NewReasonPayload("deleted"),
	})
	if err != nil {
		log.Error().Err(err).Str("saga_id", sagaID.String()).Int64("event_id", cmd.Id).
			Msg("delete event saga failed and was rolled back")
		return nil, saga.SagaError(sagaID, err)
	}

	return deleted, nil
}
