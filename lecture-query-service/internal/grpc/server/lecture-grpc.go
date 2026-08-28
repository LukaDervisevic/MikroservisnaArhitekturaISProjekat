package server

import (
	"context"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/cqrs/query"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/repo"
	lecturepb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcServer struct {
	getLectureByIDHandler           *query.GetLectureByIDHandler
	getLectureByNameHandler         *query.GetLectureByNameHandler
	listLecturesByEventIDHandler    *query.ListLecturesByEventIDHandler
	listLecturesByLecturerIDHandler *query.ListLecturesByLecturerIDHandler

	lecturepb.UnimplementedLectureServiceServer
}

func NewGrpcServer(lectureRepo repo.ILectureQueryRepo) *GrpcServer {
	return &GrpcServer{
		getLectureByIDHandler:           query.NewGetLectureByIDHandler(lectureRepo),
		getLectureByNameHandler:         query.NewGetLectureByNameHandler(lectureRepo),
		listLecturesByEventIDHandler:    query.NewListLecturesByEventIDHandler(lectureRepo),
		listLecturesByLecturerIDHandler: query.NewListLecturesByLecturerIDHandler(lectureRepo),
	}
}

func (g *GrpcServer) GetLectureByID(ctx context.Context, req *lecturepb.GetLectureByIDRequest) (*lecturepb.GetLectureByIDQueryResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecture retrieval")
	}
	lecture, err := g.getLectureByIDHandler.Handle(ctx, query.GetLectureByIDQuery{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.GetLectureByIDQueryResponse{Lecture: lectureQueryToProto(lecture)}, nil
}

func (g *GrpcServer) GetLectureByName(ctx context.Context, req *lecturepb.GetLectureByNameRequest) (*lecturepb.GetLectureByNameQueryResponse, error) {
	if req == nil || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required for lecture retrieval")
	}
	lecture, err := g.getLectureByNameHandler.Handle(ctx, query.GetLectureByNameQuery{Name: req.Name})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.GetLectureByNameQueryResponse{Lecture: lectureQueryToProto(lecture)}, nil
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
	pbLectures := make([]*lecturepb.LectureQuery, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureQueryToProto(&l)
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
	pbLectures := make([]*lecturepb.LectureQuery, len(lectures))
	for i, l := range lectures {
		pbLectures[i] = lectureQueryToProto(&l)
	}
	return &lecturepb.ListLecturesByLecturerIDResponse{
		Lectures: pbLectures, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func lectureQueryToProto(l *model.LectureQuery) *lecturepb.LectureQuery {
	if l == nil {
		return nil
	}
	return &lecturepb.LectureQuery{
		LectureId:                l.LectureID,
		LecturerId:               l.LecturerId,
		LecturerFullName:         l.LecturerFullName,
		LecturerTitle:            l.LecturerTitle,
		LecturerFieldOfExpertise: l.LecturerFieldOfExpertise,
		EventId:                  l.EventId,
		EventName:                l.EventName,
		EventCotisationPrice:     l.EventCotisationPrice,
		EventAgenda:              l.EventAgenda,
		EventType:                l.EventType,
		EventDateTime:            &timestamppb.Timestamp{Seconds: l.EventDateTime},
		LocationId:               l.LocationID,
		LocationName:             l.LocationName,
		LocationAddress:          l.LocationAddress,
		LocationCapacity:         l.LocationCapacity,
		LectureName:              l.Name,
		LectureDuration:          durationpb.New(time.Duration(l.Duration) * time.Second),
	}
}
