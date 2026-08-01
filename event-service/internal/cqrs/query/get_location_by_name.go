package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLocationByNameQuery struct {
	Name string
}

func (q GetLocationByNameQuery) Validate() error {
	if q.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type GetLocationByNameHandler struct {
	locationReadRepo repo.ILocationReadRepo
}

func NewGetLocationByNameHandler(readRepo repo.ILocationReadRepo) *GetLocationByNameHandler {
	return &GetLocationByNameHandler{locationReadRepo: readRepo}
}

func (h *GetLocationByNameHandler) Handle(ctx context.Context, q GetLocationByNameQuery) (*model.Location, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	location, err := h.locationReadRepo.GetLocationByName(ctx, q.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	return location, nil
}
