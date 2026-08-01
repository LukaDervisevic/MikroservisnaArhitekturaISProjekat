package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/cqrs/command"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/cqrs/query"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	lecturepb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type GrpcServer struct {
	db *gorm.DB

	createLectureHandler            *command.CreateLectureHandler
	updateLectureHandler            *command.UpdateLectureHandler
	deleteLectureHandler            *command.DeleteLectureHandler
	getLectureByIDHandler           *query.GetLectureByIDHandler
	getLectureByNameHandler         *query.GetLectureByNameHandler
	listLecturesByEventIDHandler    *query.ListLecturesByEventIDHandler
	listLecturesByLecturerIDHandler *query.ListLecturesByLecturerIDHandler

	lecturepb.UnimplementedLectureServiceServer
}

func NewGrpcServer(db *gorm.DB) *GrpcServer {
	lectureRepo := repo.NewLecturerRepo(db)

	return &GrpcServer{
		db: db,

		createLectureHandler:     command.NewCreateLectureHandler(lectureRepo),
		updateLocationHandler:    command.NewUpdateLocationHandler(locationRepo, locationRepo),
		deleteLocationHandler:    command.NewDeleteLocationHandler(locationRepo, locationRepo),
		getLocationByIDHandler:   query.NewGetLocationByIDHandler(locationRepo),
		getLocationByNameHandler: query.NewGetLocationByNameHandler(locationRepo),
		listLocationsHandler:     query.NewListLocationsHandler(locationRepo),

		createEventHandler:    command.NewCreateEventHandler(eventRepo, locationRepo),
		updateEventHandler:    command.NewUpdateEventHandler(eventRepo, eventRepo, locationRepo),
		deleteEventHandler:    command.NewDeleteEventHandler(eventRepo, eventRepo),
		getEventByIDHandler:   query.NewGetEventByIDHandler(eventRepo),
		getEventByNameHandler: query.NewGetEventByNameHandler(eventRepo),
		listEventsHandler:     query.NewListEventsHandler(eventRepo),
	}
}

func (g *GrpcServer) CreateLecture(ctx context.Context, req *lecturepb.CreateLectureRequest) (*lecturepb.CreateLectureResponse, error) {
	if req == nil || req.Name == "" || req.EventId == 0 || req.LecturerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "name, event_id and lecturer_id are required for lecture creation")
	}
	lecture, err := g.createLectureHandler.Handle(ctx, command.CreateLectureCommand{
		EventID: req.EventId, LecturerID: req.LecturerId, Name: req.Name, Duration: req.Duration.GetSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.CreateLectureResponse{Lecture: lectureModelToProto(lecture)}, nil
}

func (g *GrpcServer) GetLectureByID(ctx context.Context, req *lecturepb.GetLectureByIDRequest) (*lecturepb.GetLectureByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture retrieval")
	}
	lecture, err := g.getLectureByIDHandler.Handle(ctx, query.GetLectureByIDQuery{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.GetLectureByIDResponse{Lecture: lectureModelToProto(lecture)}, nil
}

func (g *GrpcServer) GetLectureByName(ctx context.Context, req *lecturepb.GetLectureByNameRequest) (*lecturepb.GetLectureByNameResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for lecture retrieval")
	}
	lecture, err := g.getLectureByNameHandler.Handle(ctx, query.GetLectureByNameQuery{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.GetLectureByNameResponse{Lecture: lectureModelToProto(lecture)}, nil
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
	lectures, totalCount, err := g.listLecturesByEventIDHandler.Handle(ctx, query.ListLecturesByEventIDQuery{
		EventID: req.EventId, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLectures := make([]*lecturepb.Lecture, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureModelToProto(&l)
	}
	return &lecturepb.ListLecturesByEventIDResponse{
		Lectures: pbLectures, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
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
	lectures, totalCount, err := g.listLecturesByLecturerIDHandler.Handle(ctx, query.ListLecturesByLecturerIDQuery{
		LecturerID: req.LecturerId, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLectures := make([]*lecturepb.Lecture, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureModelToProto(&l)
	}
	return &lecturepb.ListLecturesByLecturerIDResponse{
		Lectures: pbLectures, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLecture(ctx context.Context, req *lecturepb.UpdateLectureRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture update")
	}
	_, err := g.updateLectureHandler.Handle(ctx, command.UpdateLectureCommand{
		LectureID: req.Id, EventID: req.EventId, LecturerID: req.LecturerId, Name: req.Name, Duration: req.Duration.GetSeconds(),
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
	lecture, err := g.deleteLectureHandler.Handle(ctx, command.DeleteLectureCommand{LectureID: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.DeleteLectureResponse{Lecture: lectureModelToProto(lecture)}, nil
}

func lectureModelToProto(l *model.Lecture) *lecturepb.Lecture {
	if l == nil {
		return nil
	}
	return &lecturepb.Lecture{
		Id: l.LectureID, Lecturer: lecturerModelToProto(l.Lecturer), Event: eventModelToProto(l.Event),
		Name: l.Name, Duration: durationpb.New(time.Duration(l.Duration)),
	}
}
