package server

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service"
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	"gorm.io/gorm"
)

type GrpcServer struct {
	db              *gorm.DB
	lecturerRepo    repo.LecturerRepo
	lecturerService service.LecturerService
	lecturerpb.UnimplementedLecturerServiceServer
}

func NewGrpcServer(db *gorm.DB) *GrpcServer {
	lecturerRepo := repo.NewLecturerRepo(db)
	lecturerService := service.NewLecturerService(db, lecturerRepo)

	env := os.Getenv("ENVIRONMENT")
	var lecturerUrl string
	lecturerPort := os.Getenv("LECTURER_SERVICE_PORT")

	switch env {
	case "dev":
		lecturerUrl = fmt.Sprintf("lecturer-service:%s", lecturerPort)
	case "azure":
		lecturerAppUrl := os.Getenv("LECTURER_CONTAINER_APP_URL")
		lecturerUrl = fmt.Sprintf("%s:%s", lecturerAppUrl, lecturerPort)
	default:
		fmt.Printf("Invalid environment on lecturer grpc server")
	}
	log.Printf("lecturer url: %s", lecturerUrl)

	_, err := grpc.NewClient(lecturerUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil
	}

	return &GrpcServer{
		db:              db,
		lecturerRepo:    *lecturerRepo,
		lecturerService: *lecturerService,
	}
}

func (g *GrpcServer) CreateLecturer(ctx context.Context, req *lecturerpb.CreateLecturerRequest) (*lecturerpb.CreateLecturerResponse, error) {
	if req == nil || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required for lecturer creation")
	}

	lecturer, err := g.lecturerService.CreateLecturer(ctx, &service.CreateLecturerInput{
		FullName:         req.FullName,
		Title:            req.Title,
		FieldOfExpertise: req.FieldOfExpertise,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturerpb.CreateLecturerResponse{
		Lecturer: lecturerModelToProto(lecturer),
	}, nil
}

func (g *GrpcServer) GetLecturerByID(ctx context.Context, req *lecturerpb.GetLecturerByIDRequest) (*lecturerpb.GetLecturerByIDResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecturer retrieval")
	}

	lecturer, err := g.lecturerService.GetLecturerByID(ctx, &service.GetLecturerByIDInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturerpb.GetLecturerByIDResponse{
		Lecturer: lecturerModelToProto(lecturer),
	}, nil
}

func (g *GrpcServer) GetLecturerByName(ctx context.Context, req *lecturerpb.GetLecturerByNameRequest) (*lecturerpb.GetLecturerByNameResponse, error) {
	if req == nil || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required for lecturer retrieval")
	}

	lecturer, err := g.lecturerService.GetLecturerByName(ctx, &service.GetLecturerByNameInput{FullName: req.FullName})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturerpb.GetLecturerByNameResponse{
		Lecturer: lecturerModelToProto(lecturer),
	}, nil
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

	lecturers, totalCount, err := g.lecturerService.ListLecturers(ctx, &service.ListLecturersInput{
		Page:             page,
		PageSize:         pageSize,
		FieldOfExpertise: req.FieldOfExpertise,
		Title:            req.Title,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbLecturers := make([]*lecturerpb.Lecturer, len(lecturers))
	for i, l := range lecturers {
		pbLecturers[i] = lecturerModelToProto(&l)
	}

	return &lecturerpb.ListLecturersResponse{
		Lecturers:   pbLecturers,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
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

	lecturers, totalCount, err := g.lecturerService.ListLecturers(ctx, &service.ListLecturersInput{
		Page:             page,
		PageSize:         pageSize,
		FieldOfExpertise: req.FieldOfExpertise,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	pbLecturers := make([]*lecturerpb.Lecturer, len(lecturers))
	for i, l := range lecturers {
		pbLecturers[i] = lecturerModelToProto(&l)
	}

	return &lecturerpb.ListLecturersByFieldOfExpertiseResponse{
		Lecturers:   pbLecturers,
		TotalCount:  int32(totalCount),
		Page:        int32(page),
		PageSize:    int32(pageSize),
		HasNextPage: int64(page*pageSize) < totalCount,
	}, nil
}

func (g *GrpcServer) UpdateLecturer(ctx context.Context, req *lecturerpb.UpdateLecturerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required for lecturer update")
	}

	err := g.lecturerService.UpdateLecturer(ctx, &service.UpdateLecturerInput{
		Id:               req.Id,
		FullName:         req.FullName,
		Title:            req.Title,
		FieldOfExpertise: req.FieldOfExpertise,
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

	lecturer, err := g.lecturerService.DeleteLecturer(ctx, &service.DeleteLecturerInput{Id: req.Id})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}

	return &lecturerpb.DeleteLecturerResponse{
		Lecturer: lecturerModelToProto(lecturer),
	}, nil
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
