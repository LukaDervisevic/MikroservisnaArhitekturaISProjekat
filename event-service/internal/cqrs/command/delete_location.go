package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteLocationCommand struct {
	Id int64
}

func (c DeleteLocationCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return nil
}

type DeleteLocationHandler struct {
	locationWriteRepo repo.ILocationWriteRepo
	locationReadRepo  repo.ILocationReadRepo
}

func NewDeleteLocationHandler(writeRepo repo.ILocationWriteRepo, readRepo repo.ILocationReadRepo) *DeleteLocationHandler {
	return &DeleteLocationHandler{locationWriteRepo: writeRepo, locationReadRepo: readRepo}
}

func (h *DeleteLocationHandler) Handle(ctx context.Context, cmd DeleteLocationCommand) (*model.Location, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	location, err := h.locationReadRepo.GetLocationByID(ctx, cmd.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	if err := h.locationWriteRepo.DeleteLocation(ctx, cmd.Id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete location")
	}
	return location, nil
}
