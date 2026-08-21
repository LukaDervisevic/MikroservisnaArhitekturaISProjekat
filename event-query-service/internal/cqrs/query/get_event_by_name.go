package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetEventByNameQuery struct {
	Name string
}

func (q GetEventByNameQuery) Validate() error {
	if q.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type GetEventByNameHandler struct {
	eventReadRepo repo.IEventQueryRepo
}

func NewGetEventByNameHandler(readRepo repo.IEventQueryRepo) *GetEventByNameHandler {
	return &GetEventByNameHandler{eventReadRepo: readRepo}
}

func (h *GetEventByNameHandler) Handle(ctx context.Context, q GetEventByNameQuery) (*model.EventWithLocation, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	event, err := h.eventReadRepo.GetEventByName(ctx, q.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return event, nil
}
