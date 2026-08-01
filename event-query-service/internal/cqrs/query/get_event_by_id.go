package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetEventByIDQuery struct {
	Id int64
}

func (q GetEventByIDQuery) Validate() error {
	if q.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type GetEventByIDHandler struct {
	eventReadRepo repo.IEventQueryRepo
}

func NewGetEventByIDHandler(readRepo repo.IEventQueryRepo) *GetEventByIDHandler {
	return &GetEventByIDHandler{eventReadRepo: readRepo}
}

func (h *GetEventByIDHandler) Handle(ctx context.Context, q GetEventByIDQuery) (*model.EventWithLocation, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	event, err := h.eventReadRepo.GetEventByID(ctx, q.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return event, nil
}
