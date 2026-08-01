package server

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/cqrs/command"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/cqrs/query"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/scheduler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service" // still needed for OutboxProcessor
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type GrpcServer struct {
	db *gorm.DB

	createLecturerHandler    *command.CreateLecturerHandler
	updateLecturerHandler    *command.UpdateLecturerHandler
	deleteLecturerHandler    *command.DeleteLecturerHandler
	getLecturerByIDHandler   *query.GetLecturerByIDHandler
	getLecturerByNameHandler *query.GetLecturerByNameHandler
	listLecturersHandler     *query.ListLecturersHandler

	lecturerpb.UnimplementedLecturerServiceServer
	BrokerConn      rabbitmq.BrokerClientConn
	scheduler       *scheduler.Scheduler
	outboxProcessor *service.OutboxProcessor
}

var maxTries = 5

func isRetriable(grpcCode codes.Code) bool {
	return grpcCode != codes.InvalidArgument &&
		grpcCode != codes.DeadlineExceeded &&
		grpcCode != codes.Unavailable &&
		grpcCode != codes.NotFound &&
		grpcCode != codes.PermissionDenied
}

func NewGrpcServer(ctx context.Context, db *gorm.DB) *GrpcServer {
	lecturerRepo := repo.NewLecturerRepo(db)
	outboxRepo := repo.NewOutboxRepo(db)

	createHandler := command.NewCreateLecturerHandler(db, lecturerRepo, outboxRepo)
	updateHandler := command.NewUpdateLecturerHandler(lecturerRepo)
	deleteHandler := command.NewDeleteLecturerHandler(lecturerRepo)
	getByIDHandler := query.NewGetLecturerByIDHandler(lecturerRepo)
	getByNameHandler := query.NewGetLecturerByNameHandler(lecturerRepo)
	listHandler := query.NewListLecturersHandler(lecturerRepo)

	var lecturerBrokerConn *rabbitmq.BrokerClientConn
	var lecturerScheduler *scheduler.Scheduler
	sendQueue := os.Getenv("RABBITMQ_LECTURER_QUEUE")

	lecturerBrokerConn = rabbitmq.NewRabbitMQClientConn(ctx, os.Getenv("RABBITMQ_BROKER_URI"), nil)
	if lecturerBrokerConn == nil {
		log.Fatal().Msg("unable to connect to create rabbitmq connection")
	} else {
		lecturerBrokerConn.NewQueueRequester(ctx, lecturerBrokerConn.Connection, sendQueue)
		lecturerScheduler = scheduler.NewScheduler(10, 100)
	}

	outboxProcessor := service.NewOutboxProcessor(db, outboxRepo, lecturerBrokerConn)
	outboxProcessor.StartPoller(ctx, 2*time.Second)

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

	var loggerInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.OK:
			log.Info().Msg(fmt.Sprintf("method %s called", method))
		case codes.DeadlineExceeded:
			log.Error().Msg(fmt.Sprintf("time exceeded for method %s", method))
		}
		return err
	}

	var timeoutInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctxt, cancel := context.WithTimeout(ctx, time.Duration(70)*time.Millisecond)
		defer cancel()
		return invoker(ctxt, method, req, reply, cc, opts...)
	}

	var retryInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var err error
		for try := 0; try < maxTries; try++ {
			if try > 0 {
				jitter := rand.Intn(100)
				time.Sleep(time.Duration(math.Pow(2, float64(try)))*100*time.Millisecond + time.Duration(jitter))
				log.Warn().Msg(fmt.Sprintf("retrying method %s attempt %d/%d", method, try+1, maxTries))
			}
			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				break
			}
			st, _ := status.FromError(err)
			if !isRetriable(st.Code()) {
				break
			}
		}
		return err
	}

	grpc.WithChainUnaryInterceptor(loggerInterceptor)
	grpc.WithChainUnaryInterceptor(retryInterceptor)
	grpc.WithChainUnaryInterceptor(timeoutInterceptor)

	grpcServer := &GrpcServer{
		db: db,

		createLecturerHandler:    createHandler,
		updateLecturerHandler:    updateHandler,
		deleteLecturerHandler:    deleteHandler,
		getLecturerByIDHandler:   getByIDHandler,
		getLecturerByNameHandler: getByNameHandler,
		listLecturersHandler:     listHandler,

		scheduler:       lecturerScheduler,
		outboxProcessor: outboxProcessor,
	}
	if lecturerBrokerConn != nil {
		grpcServer.BrokerConn = *lecturerBrokerConn
	}

	return grpcServer
}

func (g *GrpcServer) CreateLecturer(ctx context.Context, req *lecturerpb.CreateLecturerRequest) (*lecturerpb.CreateLecturerResponse, error) {
	if req == nil || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required for lecturer creation")
	}
	lecturer, err := g.createLecturerHandler.Handle(ctx, command.CreateLecturerCommand{
		FullName:         req.FullName,
		Title:            req.Title,
		FieldOfExpertise: req.FieldOfExpertise,
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
		Id: req.Id, FullName: req.FullName, Title: req.Title, FieldOfExpertise: req.FieldOfExpertise,
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
		Id: l.Id, FullName: l.FullName, Title: l.Title, FieldOfExpertise: l.FieldOfExpertise,
	}
}
