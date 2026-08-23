package server

import (
	"context"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/cqrs/command"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/cqrs/query"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	outboxrepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo/outbox"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	locationpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/location"
	"gorm.io/gorm"
)

type GrpcServer struct {
	createLocationHandler    *command.CreateLocationHandler
	updateLocationHandler    *command.UpdateLocationHandler
	deleteLocationHandler    *command.DeleteLocationHandler
	getLocationByIDHandler   *query.GetLocationByIDHandler
	getLocationByNameHandler *query.GetLocationByNameHandler
	listLocationsHandler     *query.ListLocationsHandler

	createEventHandler *command.CreateEventHandler
	updateEventHandler *command.UpdateEventHandler
	deleteEventHandler *command.DeleteEventHandler

	getEventByIDHandler *query.GetEventByIDHandler

	eventpb.UnimplementedEventServiceServer
	locationpb.UnimplementedLocationServiceServer
}

func NewGrpcServer(
	db *gorm.DB,
	eventRepo *repo.EventRepo,
	locationRepo *repo.LocationRepo,
	queryBroker *rabbitmq.PublisherConn,
	outboxRepo *outboxrepo.OutboxRepo,
) *GrpcServer {
	return &GrpcServer{
		createLocationHandler:    command.NewCreateLocationHandler(db, locationRepo, outboxRepo),
		updateLocationHandler:    command.NewUpdateLocationHandler(db, locationRepo, locationRepo, outboxRepo),
		deleteLocationHandler:    command.NewDeleteLocationHandler(db, locationRepo, locationRepo, outboxRepo),
		getLocationByIDHandler:   query.NewGetLocationByIDHandler(locationRepo),
		getLocationByNameHandler: query.NewGetLocationByNameHandler(locationRepo),
		listLocationsHandler:     query.NewListLocationsHandler(locationRepo),

		createEventHandler: command.NewCreateEventHandler(db, eventRepo, locationRepo, queryBroker, outboxRepo),
		updateEventHandler: command.NewUpdateEventHandler(db, eventRepo, locationRepo, queryBroker, outboxRepo),
		deleteEventHandler: command.NewDeleteEventHandler(db, eventRepo, queryBroker, outboxRepo),

		getEventByIDHandler: query.NewGetEventByIDHandler(eventRepo),
	}
}

func (g *GrpcServer) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for location creation")
	}
	location, err := g.createLocationHandler.Handle(ctx, command.CreateLocationCommand{
		Name:     req.Name,
		Address:  req.Address,
		Capacity: req.Capacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &locationpb.CreateLocationResponse{Location: locationModelToProto(location)}, nil
}

func (g *GrpcServer) GetLocationByID(ctx context.Context, req *locationpb.GetLocationByIDRequest) (*locationpb.GetLocationByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for location retrieval")
	}
	location, err := g.getLocationByIDHandler.Handle(ctx, query.GetLocationByIDQuery{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &locationpb.GetLocationByIDResponse{Location: locationModelToProto(location)}, nil
}

func (g *GrpcServer) GetLocationByName(ctx context.Context, req *locationpb.GetLocationByNameRequest) (*locationpb.GetLocationByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for location retrieval")
	}
	location, err := g.getLocationByNameHandler.Handle(ctx, query.GetLocationByNameQuery{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &locationpb.GetLocationByNameResponse{Location: locationModelToProto(location)}, nil
}

func (g *GrpcServer) ListLocations(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
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
	locations, totalCount, err := g.listLocationsHandler.Handle(ctx, query.ListLocationsQuery{
		Page:        page,
		PageSize:    pageSize,
		MinCapacity: req.MinCapacity,
		MaxCapacity: req.MaxCapacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLocations := make([]*locationpb.Location, len(locations))
	for i, l := range locations {
		pbLocations[i] = locationModelToProto(&l)
	}
	return &locationpb.ListLocationsResponse{
		Locations: pbLocations, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) ListLocationsByMinCapacity(ctx context.Context, req *locationpb.ListLocationsByMinCapacityRequest) (*locationpb.ListLocationsByMinCapacityResponse, error) {
	if req == nil || req.MinCapacity == 0 {
		return nil, status.Error(codes.InvalidArgument, "min_capacity is required")
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	locations, totalCount, err := g.listLocationsHandler.Handle(ctx, query.ListLocationsQuery{
		Page: page, PageSize: pageSize, MinCapacity: req.MinCapacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLocations := make([]*locationpb.Location, len(locations))
	for i, l := range locations {
		pbLocations[i] = locationModelToProto(&l)
	}
	return &locationpb.ListLocationsByMinCapacityResponse{
		Locations: pbLocations, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for location update")
	}
	err := g.updateLocationHandler.Handle(ctx, command.UpdateLocationCommand{
		Id: req.Id, Name: req.Name, Address: req.Address, Capacity: req.Capacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &emptypb.Empty{}, nil
}

func (g *GrpcServer) DeleteLocation(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for location deletion")
	}
	location, err := g.deleteLocationHandler.Handle(ctx, command.DeleteLocationCommand{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &locationpb.DeleteLocationResponse{Location: locationModelToProto(location)}, nil
}

func (g *GrpcServer) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req == nil || req.Name == "" || req.Agenda == "" || req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "name, agenda and type are required for event creation")
	}

	event, err := g.createEventHandler.Handle(ctx, command.CreateEventCommand{
		Name: req.Name, CotisationPrice: req.CotisationPrice, Agenda: req.Agenda, Type: req.Type,
		DateTime: req.DateTime.GetSeconds(), LocationID: req.LocationId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &eventpb.CreateEventResponse{Event: eventModelToProto(event)}, nil
}

func (g *GrpcServer) GetEventByID(ctx context.Context, req *eventpb.GetEventByIdRequest) (*eventpb.GetEventByIdQueryResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event retrieval")
	}
	event, err := g.getEventByIDHandler.Handle(ctx, query.GetEventByIDQuery{Id: req.Id})
	if err != nil {
		return nil, err
	}
	return &eventpb.GetEventByIdQueryResponse{EventWithLocation: eventWithLocationModelToProto(event)}, nil
}

func (g *GrpcServer) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event update")
	}
	_, err := g.updateEventHandler.Handle(ctx, command.UpdateEventCommand{
		Id: req.Id, Name: req.Name, CotisationPrice: req.CotisationPrice, Agenda: req.Agenda,
		Type: req.Type, DateTime: req.DateTime.GetSeconds(), LocationID: req.LocationId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &emptypb.Empty{}, nil
}

func (g *GrpcServer) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event deletion")
	}
	event, err := g.deleteEventHandler.Handle(ctx, command.DeleteEventCommand{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &eventpb.DeleteEventResponse{Event: eventModelToProto(event)}, nil
}

func locationModelToProto(l *model.Location) *locationpb.Location {
	if l == nil {
		return nil
	}
	return &locationpb.Location{Id: l.Id, Name: l.Name, Address: l.Address, Capacity: l.Capacity}
}

func eventModelToProto(e *model.Event) *eventpb.Event {
	if e == nil {
		return nil
	}
	return &eventpb.Event{
		Id: e.Id, Name: e.Name, CotisationPrice: e.CotisationPrice, Agenda: e.Agenda, Type: e.Type,
		DateTime: timestamppb.New(time.Unix(e.DateTime, 0)),
		Location: locationModelToProto(e.Location),
	}
}

func eventWithLocationModelToProto(e *model.EventWithLocation) *eventpb.EventWithLocation {
	if e == nil {
		return nil
	}
	return &eventpb.EventWithLocation{
		EventId:              e.EventId,
		EventName:            e.EventName,
		EventCotisationPrice: e.EventCotisationPrice,
		EventAgenda:          e.EventAgenda,
		EventType:            e.EventType,
		EventDateTime:        timestamppb.New(time.Unix(e.EventDateTime, 0)),
		LocationId:           e.LocationID,
		LocationName:         e.LocationName,
		LocationAddress:      e.LocationAddress,
		LocationCapacity:     e.LocationCapacity,
	}
}
