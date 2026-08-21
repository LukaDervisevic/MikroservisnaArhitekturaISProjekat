package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListLocationsQuery struct {
	Page        int
	PageSize    int
	MinCapacity int64
	MaxCapacity int64
}

func (q ListLocationsQuery) Validate() error {
	if q.Page < 0 {
		return status.Error(codes.InvalidArgument, "page must be >= 0")
	}
	if q.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page size must be > 0")
	}
	return nil
}

type ListLocationsHandler struct {
	locationReadRepo repo.ILocationReadRepo
}

func NewListLocationsHandler(readRepo repo.ILocationReadRepo) *ListLocationsHandler {
	return &ListLocationsHandler{locationReadRepo: readRepo}
}

func (h *ListLocationsHandler) Handle(ctx context.Context, q ListLocationsQuery) ([]model.Location, int64, error) {
	if err := q.Validate(); err != nil {
		return nil, 0, err
	}
	locations, totalCount, err := h.locationReadRepo.ListLocations(ctx, repo.ListLocationsFilter{
		Page:        q.Page,
		PageSize:    q.PageSize,
		MinCapacity: q.MinCapacity,
		MaxCapacity: q.MaxCapacity,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list locations")
	}
	return locations, totalCount, nil
}
