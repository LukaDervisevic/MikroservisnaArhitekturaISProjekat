package query

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ListEventsQuery struct {
	Page     int
	PageSize int
	Type     string
	FromDate int64
	ToDate   int64
}

func (q ListEventsQuery) Validate() error {
	if q.Page < 0 {
		return status.Error(codes.InvalidArgument, "page must be >= 0")
	}
	if q.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page size must be > 0")
	}
	return nil
}

type ListEventsHandler struct {
	eventReadRepo repo.IEventQueryRepo
}

func NewListEventsHandler(readRepo repo.IEventQueryRepo) *ListEventsHandler {
	return &ListEventsHandler{eventReadRepo: readRepo}
}

func (h *ListEventsHandler) Handle(ctx context.Context, q ListEventsQuery) ([]model.EventWithLocation, int64, error) {
	if err := q.Validate(); err != nil {
		return nil, 0, err
	}
	events, totalCount, err := h.eventReadRepo.ListEvents(ctx, repo.ListEventsFilter{
		Page:     q.Page,
		PageSize: q.PageSize,
		Type:     q.Type,
		FromDate: q.FromDate,
		ToDate:   q.ToDate,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list events")
	}
	return events, totalCount, nil
}
