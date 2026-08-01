package server

import (
	"context"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/cqrs/query"
	model2 "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/repo"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type GrpcServer struct {
	db *gorm.DB

	getEventByIDHandler   *query.GetEventByIDHandler
	getEventByNameHandler *query.GetEventByNameHandler
	listEventsHandler     *query.ListEventsHandler

	eventpb.UnimplementedEventServiceServer
}

func NewGrpcServer(db *gorm.DB) *GrpcServer {
	eventRepo := repo.NewEventRepo(db)

	return &GrpcServer{
		db: db,

		getEventByIDHandler:   query.NewGetEventByIDHandler(eventRepo),
		getEventByNameHandler: query.NewGetEventByNameHandler(eventRepo),
		listEventsHandler:     query.NewListEventsHandler(eventRepo),
	}
}

func (g *GrpcServer) GetEventByID(ctx context.Context, req *eventpb.GetEventByIdRequest) (*eventpb.GetEventByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event retrieval")
	}
	event, err := g.getEventByIDHandler.Handle(ctx, query.GetEventByIDQuery{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &eventpb.GetEventByIDResponse{Event: eventWithLocationModelToProto(event)}, nil
}

func (g *GrpcServer) GetEventByName(ctx context.Context, req *eventpb.GetEventByNameRequest) (*eventpb.GetEventByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for event retrieval")
	}
	event, err := g.getEventByNameHandler.Handle(ctx, query.GetEventByNameQuery{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &eventpb.GetEventByNameResponse{Event: eventWithLocationModelToProto(event)}, nil
}

func (g *GrpcServer) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	events, totalCount, err := g.listEventsHandler.Handle(ctx, query.ListEventsQuery{
		Page: page, PageSize: pageSize, Type: req.Type,
		FromDate: req.FromDate.GetSeconds(), ToDate: req.ToDate.GetSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbEvents := make([]*eventpb.Event, len(events))
	for i, e := range events {
		pbEvents[i] = eventWithLocationModelToProto(&e)
	}
	return &eventpb.ListEventsResponse{
		Events: pbEvents, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) ListEventsByType(ctx context.Context, req *eventpb.ListEventsByTypeRequest) (*eventpb.ListEventsByTypeResponse, error) {
	if req == nil || req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	events, totalCount, err := g.listEventsHandler.Handle(ctx, query.ListEventsQuery{
		Page: page, PageSize: pageSize, Type: req.Type,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbEvents := make([]*eventpb.Event, len(events))
	for i, e := range events {
		pbEvents[i] = eventWithLocationModelToProto(&e)
	}
	return &eventpb.ListEventsByTypeResponse{
		Events: pbEvents, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func eventWithLocationModelToProto(e *model2.EventWithLocation) *eventpb.Event {
	if e == nil {
		return nil
	}
	panic("TODO: implement eventWithLocationModelToProto")
}
