package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetLocationByIDQuery struct {
	Id int64
}

func (q GetLocationByIDQuery) Validate() error {
	if q.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type GetLocationByIDHandler struct {
	locationReadRepo repo.ILocationReadRepo
}

func NewGetLocationByIDHandler(readRepo repo.ILocationReadRepo) *GetLocationByIDHandler {
	return &GetLocationByIDHandler{locationReadRepo: readRepo}
}

func (h *GetLocationByIDHandler) Handle(ctx context.Context, q GetLocationByIDQuery) (*model.Location, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	location, err := h.locationReadRepo.GetLocationByID(ctx, q.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	return location, nil
}
