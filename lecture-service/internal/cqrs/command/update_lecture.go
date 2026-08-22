package command

import (
	"context"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/service/saga"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// sagaTimeout bounds how long this service, as a saga initiator, waits for the
// chain to report back.
const sagaTimeout = 20 * time.Second

type UpdateLectureCommand struct {
	LectureID  int64
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

func (c UpdateLectureCommand) Validate() error {
	if c.LectureID <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	return nil
}

type UpdateLectureHandler struct {
	db            *gorm.DB
	lectureRepo   repo.ILectureRepo
	eventRepo     repo.IEventReadRepo
	publisherConn *rabbitmq.PublisherConn
	sagaReplies   *saga.SagaReplyRegistry
}

func NewUpdateLectureHandler(
	db *gorm.DB,
	lectureRepo repo.ILectureRepo,
	eventRepo repo.IEventReadRepo,
	brokerConn *rabbitmq.PublisherConn,
	sagaReplies *saga.SagaReplyRegistry,
) *UpdateLectureHandler {
	return &UpdateLectureHandler{
		db:            db,
		lectureRepo:   lectureRepo,
		eventRepo:     eventRepo,
		publisherConn: brokerConn,
		sagaReplies:   sagaReplies,
	}
}

func (h *UpdateLectureHandler) Handle(
	ctx context.Context,
	cmd UpdateLectureCommand) (*model.Lecture, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	lecture, err := h.lectureRepo.GetLectureByID(ctx, cmd.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if cmd.EventID != 0 && cmd.EventID != lecture.EventID {
		event, err := h.eventRepo.GetEventByID(ctx, cmd.EventID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify event")
		}
		if event == nil {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		lecture.EventID = cmd.EventID
	}
	lecture.LecturerID = cmd.LecturerID
	lecture.Name = cmd.Name
	lecture.Duration = cmd.Duration

	lectureQuery := mapper.MapLectureToQuery(lecture)
	if lectureQuery == nil {
		return nil, status.Error(codes.Internal, "lecture is missing event or lecturer data")
	}

	// Same shape as the lecturer saga: hold the local write until the read model
	// confirms it applied the change.
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sagaID := uuid.New()
		ch := h.sagaReplies.Register(sagaID)
		defer h.sagaReplies.Unregister(sagaID)

		if err := h.publisherConn.PublishSaga(
			ctx,
			os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE"),
			sagaID,
			"UpdateLectureQuerySAGA",
			[]*model.LectureQuery{lectureQuery},
		); err != nil {
			log.Error().Err(err).Msgf("failed to dispatch saga %s for lecture %d", sagaID, lecture.LectureID)
			return status.Error(codes.Internal, "failed to dispatch lecture saga")
		}

		select {
		case sagaErr := <-ch:
			if sagaErr != nil {
				log.Warn().Err(sagaErr).Msgf("saga %s rolled back for lecture %d", sagaID, lecture.LectureID)
				return status.Error(codes.Internal, "saga rolled back: "+sagaErr.Error())
			}
		case <-time.After(sagaTimeout):
			log.Warn().Msgf("saga %s timed out for lecture %d", sagaID, lecture.LectureID)
			return status.Error(codes.DeadlineExceeded, "saga reply timeout")
		}

		if err := h.lectureRepo.UpdateLecture(ctx, lecture); err != nil {
			return status.Error(codes.Internal, "failed to update lecture")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return lecture, nil
}
