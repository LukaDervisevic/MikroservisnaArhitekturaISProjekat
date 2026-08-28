package server

import (
	"context"
	"fmt"
	"os"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/cqrs/command"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/cqrs/query"
	outbox2 "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo/outbox"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/saga"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type GrpcServer struct {
	createLecturerHandler    *command.CreateLecturerHandler
	updateLecturerHandler    *command.UpdateLecturerHandler
	deleteLecturerHandler    *command.DeleteLecturerHandler
	getLecturerByIDHandler   *query.GetLecturerByIDHandler
	getLecturerByNameHandler *query.GetLecturerByNameHandler
	listLecturersHandler     *query.ListLecturersHandler

	lecturerpb.UnimplementedLecturerServiceServer
}

var maxTries = 5

func isRetriable(grpcCode codes.Code) bool {
	return grpcCode != codes.InvalidArgument &&
		grpcCode != codes.DeadlineExceeded &&
		grpcCode != codes.Unavailable &&
		grpcCode != codes.NotFound &&
		grpcCode != codes.PermissionDenied
}

func NewGrpcServer(
	db *gorm.DB,
	lecturerRepo *repo.LecturerRepo,
	outboxRepo *outbox2.OutboxRepo,
	publisherConn *rabbitmq.PublisherConn,
	sagaReplies *saga.SagaReplyRegistry,
) *GrpcServer {
	createHandler := command.NewCreateLecturerHandler(db, lecturerRepo, outboxRepo)
	updateHandler := command.NewUpdateLecturerHandler(lecturerRepo, db, publisherConn, sagaReplies)
	deleteHandler := command.NewDeleteLecturerHandler(lecturerRepo)
	getByIDHandler := query.NewGetLecturerByIDHandler(lecturerRepo)
	getByNameHandler := query.NewGetLecturerByNameHandler(lecturerRepo)
	listHandler := query.NewListLecturersHandler(lecturerRepo)

	env := os.Getenv("ENVIRONMENT")
	var lecturerUrl string
	lecturerPort := os.Getenv("LECTURER_SERVICE_PORT")

	switch env {
	case "local":
		lecturerUrl = fmt.Sprintf("localhost:%s", lecturerPort)
	case "dev":
		lecturerUrl = fmt.Sprintf("lecturer-service:%s", lecturerPort)
	case "azure":
		lecturerAppUrl := os.Getenv("LECTURER_CONTAINER_APP_URL")
		lecturerUrl = fmt.Sprintf("%s:%s", lecturerAppUrl, lecturerPort)
	default:
		fmt.Printf("Invalid environment on lecturer grpc server")
	}
	log.Info().Msg(fmt.Sprintf("lecturer gRPC server url: %s", lecturerUrl))

	return &GrpcServer{
		createLecturerHandler:    createHandler,
		updateLecturerHandler:    updateHandler,
		deleteLecturerHandler:    deleteHandler,
		getLecturerByIDHandler:   getByIDHandler,
		getLecturerByNameHandler: getByNameHandler,
		listLecturersHandler:     listHandler,
	}
}

func (g *GrpcServer) CreateLecturer(ctx context.Context, req *lecturerpb.CreateLecturerRequest) (*lecturerpb.CreateLecturerResponse, error) {
	if req == nil || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required for lecturer creation")
	}
	lecturer, err := g.createLecturerHandler.Handle(ctx, command.CreateLecturerCommand{
		FullName:         req.FullName,
		Title:            req.Title,
		FieldOfExpertise: req.FieldOfExpertise,
		Email:            req.Email,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturerpb.CreateLecturerResponse{Lecturer: lecturerModelToProto(lecturer)}, nil
}

func (g *GrpcServer) GetLecturerByID(ctx context.Context, req *lecturerpb.GetLecturerByIDRequest) (*lecturerpb.GetLecturerByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecturer retrieval")
	}
	lecturer, err := g.getLecturerByIDHandler.Handle(ctx, query.GetLecturerByIDQuery{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturerpb.GetLecturerByIDResponse{Lecturer: lecturerModelToProto(lecturer)}, nil
}

func (g *GrpcServer) GetLecturerByName(ctx context.Context, req *lecturerpb.GetLecturerByNameRequest) (*lecturerpb.GetLecturerByNameResponse, error) {
	if req == nil || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required for lecturer retrieval")
	}
	lecturer, err := g.getLecturerByNameHandler.Handle(ctx, query.GetLecturerByNameQuery{FullName: req.FullName})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturerpb.GetLecturerByNameResponse{Lecturer: lecturerModelToProto(lecturer)}, nil
}

func (g *GrpcServer) ListLecturers(ctx context.Context, req *lecturerpb.ListLecturersRequest) (*lecturerpb.ListLecturersResponse, error) {
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
	lecturers, totalCount, err := g.listLecturersHandler.Handle(ctx, query.ListLecturersQuery{
		Page: page, PageSize: pageSize, FieldOfExpertise: req.FieldOfExpertise, Title: req.Title,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLecturers := make([]*lecturerpb.Lecturer, len(lecturers))
	for i, l := range lecturers {
		pbLecturers[i] = lecturerModelToProto(&l)
	}
	return &lecturerpb.ListLecturersResponse{
		Lecturers: pbLecturers, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) ListLecturersByFieldOfExpertise(ctx context.Context, req *lecturerpb.ListLecturersByFieldOfExpertiseRequest) (*lecturerpb.ListLecturersByFieldOfExpertiseResponse, error) {
	if req == nil || req.FieldOfExpertise == "" {
		return nil, status.Error(codes.InvalidArgument, "field_of_expertise is required")
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
	lecturers, totalCount, err := g.listLecturersHandler.Handle(ctx, query.ListLecturersQuery{
		Page: page, PageSize: pageSize, FieldOfExpertise: req.FieldOfExpertise,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	pbLecturers := make([]*lecturerpb.Lecturer, len(lecturers))
	for i, l := range lecturers {
		pbLecturers[i] = lecturerModelToProto(&l)
	}
	return &lecturerpb.ListLecturersByFieldOfExpertiseResponse{
		Lecturers: pbLecturers, TotalCount: int32(totalCount), Page: int32(page), PageSize: int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLecturer(ctx context.Context, req *lecturerpb.UpdateLecturerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecturer update")
	}
	err := g.updateLecturerHandler.Handle(ctx, command.UpdateLecturerCommand{
		Id: req.Id, FullName: req.FullName, Title: req.Title, FieldOfExpertise: req.FieldOfExpertise, Email: req.Email,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &emptypb.Empty{}, nil
}

func (g *GrpcServer) DeleteLecturer(ctx context.Context, req *lecturerpb.DeleteLecturerRequest) (*lecturerpb.DeleteLecturerResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecturer deletion")
	}
	lecturer, err := g.deleteLecturerHandler.Handle(ctx, command.DeleteLecturerCommand{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturerpb.DeleteLecturerResponse{Lecturer: lecturerModelToProto(lecturer)}, nil
}

func lecturerModelToProto(l *model.Lecturer) *lecturerpb.Lecturer {
	if l == nil {
		return nil
	}
	return &lecturerpb.Lecturer{
		Id: l.Id, FullName: l.FullName, Title: l.Title, FieldOfExpertise: l.FieldOfExpertise, Email: l.Email,
	}
}
