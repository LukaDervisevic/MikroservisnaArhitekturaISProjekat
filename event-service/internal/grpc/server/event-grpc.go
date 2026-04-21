package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/service"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	lecturepb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	locationpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/location"
	"gorm.io/gorm"
)

type GrpcServer struct {
	db           *gorm.DB
	eventRepo    repo.EventRepo
	eventService service.EventManagementService
	eventpb.UnimplementedEventServiceServer
}

func NewGrpcServer(db *gorm.DB) *GrpcServer {
	eventRepo := repo.NewEventRepo(db)
	eventService := service.NewEventManagementService(db, eventRepo)

	env := os.Getenv("ENVIRONMENT")
	var eventUrl string
	eventPort := os.Getenv("EVENT_SERVICE_PORT")

	switch env {
	case "dev":
		eventUrl = fmt.Sprintf("event-service:%s", eventPort)
	case "azure":
		eventAppUrl := os.Getenv("EVENT_CONTAINER_APP_URL")
		eventUrl = fmt.Sprintf("%s:%s", eventAppUrl, eventPort)
	default:
		fmt.Printf("Invalid environment on event grpc server")
	}
	log.Printf("event url: %s", eventUrl)

	_, err := grpc.NewClient(eventUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil
	}

	return &GrpcServer{
		db:           db,
		eventRepo:    *eventRepo,
		eventService: *eventService,
	}
}

func (g *GrpcServer) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for location creation")
	}

	location, err := g.eventService.CreateLocation(ctx, &service.CreateLocationInput{
		Name:     req.Name,
		Address:  req.Address,
		Capacity: req.Capacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &locationpb.CreateLocationResponse{
		Location: locationModelToProto(location),
	}, nil
}

func (g *GrpcServer) GetLocationByID(ctx context.Context, req *locationpb.GetLocationByIDRequest) (*locationpb.GetLocationByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for location retrieval")
	}

	location, err := g.eventService.GetLocationByID(ctx, &service.GetLocationByIDInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &locationpb.GetLocationByIDResponse{
		Location: locationModelToProto(location),
	}, nil
}

func (g *GrpcServer) GetLocationByName(ctx context.Context, req *locationpb.GetLocationByNameRequest) (*locationpb.GetLocationByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for location retrieval")
	}

	location, err := g.eventService.GetLocationByName(ctx, &service.GetLocationByNameInput{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &locationpb.GetLocationByNameResponse{
		Location: locationModelToProto(location),
	}, nil
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

	locations, totalCount, err := g.eventService.ListLocations(ctx, &service.ListLocationsInput{
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
		Locations:   pbLocations,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
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

	locations, totalCount, err := g.eventService.ListLocations(ctx, &service.ListLocationsInput{
		Page:        page,
		PageSize:    pageSize,
		MinCapacity: req.MinCapacity,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbLocations := make([]*locationpb.Location, len(locations))
	for i, l := range locations {
		pbLocations[i] = locationModelToProto(&l)
	}

	return &locationpb.ListLocationsByMinCapacityResponse{
		Locations:   pbLocations,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for location update")
	}

	err := g.eventService.UpdateLocation(ctx, &service.UpdateLocationInput{
		Id:       req.Id,
		Name:     req.Name,
		Address:  req.Address,
		Capacity: req.Capacity,
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

	location, err := g.eventService.DeleteLocation(ctx, &service.DeleteLocationInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &locationpb.DeleteLocationResponse{
		Location: locationModelToProto(location),
	}, nil
}

func (g *GrpcServer) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req == nil || req.Name == "" || req.Agenda == "" || req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "name, agenda and type are required for event creation")
	}

	event, err := g.eventService.CreateEvent(ctx, &service.CreateEventInput{
		Name:            req.Name,
		CotisationPrice: req.CotisationPrice,
		Agenda:          req.Agenda,
		Type:            req.Type,
		DateTime:        req.DateTime.GetSeconds(),
		LocationID:      req.Location.Id,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &eventpb.CreateEventResponse{
		Event: eventModelToProto(event),
	}, nil
}

func (g *GrpcServer) GetEventByID(ctx context.Context, req *eventpb.GetEventByIdRequest) (*eventpb.GetEventByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event retrieval")
	}

	event, err := g.eventService.GetEventByID(ctx, &service.GetEventByIDInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &eventpb.GetEventByIDResponse{
		Event: eventModelToProto(event),
	}, nil
}

func (g *GrpcServer) GetEventByName(ctx context.Context, req *eventpb.GetEventByNameRequest) (*eventpb.GetEventByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for event retrieval")
	}

	event, err := g.eventService.GetEventByName(ctx, &service.GetEventByNameInput{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &eventpb.GetEventByNameResponse{
		Event: eventModelToProto(event),
	}, nil
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

	events, totalCount, err := g.eventService.ListEvents(ctx, &service.ListEventsInput{
		Page:       page,
		PageSize:   pageSize,
		Type:       req.Type,
		FromDate:   req.FromDate.GetSeconds(),
		ToDate:     req.ToDate.GetSeconds(),
		LocationID: req.LocationId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbEvents := make([]*eventpb.Event, len(events))
	for i, e := range events {
		pbEvents[i] = eventModelToProto(&e)
	}

	return &eventpb.ListEventsResponse{
		Events:      pbEvents,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
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

	events, totalCount, err := g.eventService.ListEvents(ctx, &service.ListEventsInput{
		Page:     page,
		PageSize: pageSize,
		Type:     req.Type,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbEvents := make([]*eventpb.Event, len(events))
	for i, e := range events {
		pbEvents[i] = eventModelToProto(&e)
	}

	return &eventpb.ListEventsByTypeResponse{
		Events:      pbEvents,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for event update")
	}

	_, err := g.eventService.UpdateEvent(ctx, &service.UpdateEventInput{
		Id:              req.Id,
		Name:            req.Name,
		CotisationPrice: req.CotisationPrice,
		Agenda:          req.Agenda,
		Type:            req.Type,
		DateTime:        req.DateTime.GetSeconds(),
		LocationID:      req.Location.Id,
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

	event, err := g.eventService.DeleteEvent(ctx, &service.DeleteEventInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &eventpb.DeleteEventResponse{
		Event: eventModelToProto(event),
	}, nil
}

func (g *GrpcServer) CreateLecture(ctx context.Context, req *lecturepb.CreateLectureRequest) (*lecturepb.CreateLectureResponse, error) {
	if req == nil || req.Name == "" || req.EventId == 0 || req.LecturerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "name, event_id and lecturer_id are required for lecture creation")
	}

	lecture, err := g.eventService.CreateLecture(ctx, &service.CreateLectureInput{
		EventID:    req.EventId,
		LecturerID: req.LecturerId,
		Name:       req.Name,
		Duration:   req.Duration.GetSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturepb.CreateLectureResponse{
		Lecture: lectureModelToProto(lecture),
	}, nil
}

func (g *GrpcServer) GetLectureByID(ctx context.Context, req *lecturepb.GetLectureByIDRequest) (*lecturepb.GetLectureByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture retrieval")
	}

	lecture, err := g.eventService.GetLectureByID(ctx, &service.GetLectureByIDInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturepb.GetLectureByIDResponse{
		Lecture: lectureModelToProto(lecture),
	}, nil
}

func (g *GrpcServer) GetLectureByName(ctx context.Context, req *lecturepb.GetLectureByNameRequest) (*lecturepb.GetLectureByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for lecture retrieval")
	}

	lecture, err := g.eventService.GetLectureByName(ctx, &service.GetLectureByNameInput{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturepb.GetLectureByNameResponse{
		Lecture: lectureModelToProto(lecture),
	}, nil
}

func (g *GrpcServer) ListLecturesByEventID(ctx context.Context, req *lecturepb.ListLecturesByEventIDRequest) (*lecturepb.ListLecturesByEventIDResponse, error) {
	if req == nil || req.EventId == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	lectures, totalCount, err := g.eventService.ListLecturesByEventID(ctx, &service.ListLecturesByEventIDInput{
		EventID:  req.EventId,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbLectures := make([]*lecturepb.Lecture, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureModelToProto(&l)
	}

	return &lecturepb.ListLecturesByEventIDResponse{
		Lectures:    pbLectures,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) ListLecturesByLecturerID(ctx context.Context, req *lecturepb.ListLecturesByLecturerIDRequest) (*lecturepb.ListLecturesByLecturerIDResponse, error) {
	if req == nil || req.LecturerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "lecturer_id is required")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	lectures, totalCount, err := g.eventService.ListLecturesByLecturerID(ctx, &service.ListLecturesByLecturerIDInput{
		LecturerID: req.LecturerId,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbLectures := make([]*lecturepb.Lecture, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureModelToProto(&l)
	}

	return &lecturepb.ListLecturesByLecturerIDResponse{
		Lectures:    pbLectures,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLecture(ctx context.Context, req *lecturepb.UpdateLectureRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture update")
	}

	_, err := g.eventService.UpdateLecture(ctx, &service.UpdateLectureInput{
		LectureID:  req.Id,
		EventID:    req.EventId,
		LecturerID: req.LecturerId,
		Name:       req.Name,
		Duration:   req.Duration.GetSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &emptypb.Empty{}, nil
}

func (g *GrpcServer) DeleteLecture(ctx context.Context, req *lecturepb.DeleteLectureRequest) (*lecturepb.DeleteLectureResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture deletion")
	}

	lecture, err := g.eventService.DeleteLecture(ctx, &service.DeleteLectureInput{LectureID: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturepb.DeleteLectureResponse{
		Lecture: lectureModelToProto(lecture),
	}, nil
}

func locationModelToProto(l *model.Location) *locationpb.Location {
	if l == nil {
		return nil
	}
	return &locationpb.Location{
		Id:       l.Id,
		Name:     l.Name,
		Address:  l.Address,
		Capacity: l.Capacity,
	}
}

func eventModelToProto(e *model.Event) *eventpb.Event {
	if e == nil {
		return nil
	}
	pb := &eventpb.Event{
		Id:              e.Id,
		Name:            e.Name,
		CotisationPrice: e.CotisationPrice,
		Agenda:          e.Agenda,
		Type:            e.Type,
		DateTime:        timestamppb.New(time.Unix(e.DateTime, 0)),
		Location:        locationModelToProto(e.Location),
	}
	return pb
}

func lecturerModelToProto(l *model.Lecturer) *lecturerpb.Lecturer {
	if l == nil {
		return nil
	}
	return &lecturerpb.Lecturer{
		Id:               l.Id,
		FullName:         l.FullName,
		Title:            l.Title,
		FieldOfExpertise: l.FieldOfExpertise,
	}
}

func lectureModelToProto(l *model.Lecture) *lecturepb.Lecture {
	if l == nil {
		return nil
	}
	return &lecturepb.Lecture{
		Id:       l.LectureID,
		Lecturer: lecturerModelToProto(l.Lecturer),
		Event:    eventModelToProto(l.Event),
		Name:     l.Name,
		Duration: durationpb.New(time.Duration(l.Duration)),
	}
}
