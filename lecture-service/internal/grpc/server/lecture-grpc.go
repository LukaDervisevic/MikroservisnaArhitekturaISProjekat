package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/cqrs/command"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/service/saga"
	eventpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/event"
	lecturepb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecture"
	lecturerpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/lecturer"
	locationpb "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/proto/location"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type GrpcServer struct {
	createLectureHandler *command.CreateLectureHandler
	updateLectureHandler *command.UpdateLectureHandler
	deleteLectureHandler *command.DeleteLectureHandler

	lecturepb.UnimplementedLectureServiceServer
	lecturerClient lecturerpb.LecturerServiceClient
	eventClient    eventpb.EventServiceClient
}

func NewGrpcServer(
	db *gorm.DB,
	publisherConn *rabbitmq.PublisherConn,
	lecturerClient lecturerpb.LecturerServiceClient,
	eventClient eventpb.EventServiceClient,
	sagaReplies *saga.SagaReplyRegistry,
	lectureRepo *repo.LectureRepo,
	eventRepo *repo.EventRepo,
	lecturerRepo *repo.LecturerRepo,
	mailPublisher *rabbitmq.MailPublisher,
) *GrpcServer {

	return &GrpcServer{
		createLectureHandler: command.NewCreateLectureHandler(db, lectureRepo, eventRepo, lecturerRepo, publisherConn, mailPublisher),
		updateLectureHandler: command.NewUpdateLectureHandler(db, lectureRepo, eventRepo, lecturerRepo, publisherConn, sagaReplies),
		deleteLectureHandler: command.NewDeleteLectureHandler(db, lectureRepo, publisherConn),
		eventClient:          eventClient,
		lecturerClient:       lecturerClient,
	}
}

func (g *GrpcServer) CreateLecture(ctx context.Context, req *lecturepb.CreateLectureRequest) (*lecturepb.CreateLectureResponse, error) {
	if req == nil || req.Name == "" || req.EventId == 0 || req.LecturerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "name, event_id and lecturer_id are required for lecture creation")
	}
	lecturerValidChan := make(chan error, 1)
	eventValidChan := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		res, err := g.lecturerClient.GetLecturerByID(ctx, &lecturerpb.GetLecturerByIDRequest{Id: req.LecturerId})
		if err != nil || res == nil {
			lecturerValidChan <- err
			return
		}
		lecturerValidChan <- nil
	})
	wg.Go(func() {
		res, err := g.eventClient.GetEventByID(ctx, &eventpb.GetEventByIdRequest{Id: req.EventId})
		if err != nil || res == nil {
			eventValidChan <- err
			return
		}
		eventValidChan <- nil
	})
	wg.Wait()

	if err := <-lecturerValidChan; err != nil {
		log.Error().Err(err).Str("code", status.Code(err).String()).Msg("lecture check failed")
		return nil, status.Error(codes.Internal, "lecturer doesn't exist")
	}
	if err := <-eventValidChan; err != nil {
		log.Error().Err(err).Str("code", status.Code(err).String()).Msg("event check failed")
		return nil, status.Error(codes.Internal, "event doesn't exist")
	}

	lecture, err := g.createLectureHandler.Handle(ctx, command.CreateLectureCommand{
		EventID: req.EventId, LecturerID: req.LecturerId, Name: req.Name, Duration: req.Duration.GetSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("%v", err))
	}
	return &lecturepb.CreateLectureResponse{Lecture: lectureModelToProto(lecture)}, nil
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
		Name: l.Name, Duration: durationpb.New(time.Duration(l.Duration) * time.Second),
	}
}

func lecturerModelToProto(l *model.Lecturer) *lecturerpb.Lecturer {
	if l == nil {
		return nil
	}
	return &lecturerpb.Lecturer{
		Id: l.Id, FullName: l.FullName, Title: l.Title, FieldOfExpertise: l.FieldOfExpertise,
	}
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

func locationModelToProto(l *model.Location) *locationpb.Location {
	if l == nil {
		return nil
	}
	return &locationpb.Location{Id: l.Id, Name: l.Name, Address: l.Address, Capacity: l.Capacity}
}
