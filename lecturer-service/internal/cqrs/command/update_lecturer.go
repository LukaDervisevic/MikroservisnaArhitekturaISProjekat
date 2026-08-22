package command

import (
	"context"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/saga"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UpdateLecturerCommand struct {
	Id               int64
	FullName         string
	Title            string
	FieldOfExpertise string
}

func (c UpdateLecturerCommand) Validate() error {
	if c.Id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if c.FullName == "" {
		return status.Error(codes.InvalidArgument, "full name is required")
	}
	if c.Title == "" {
		return status.Error(codes.InvalidArgument, "title is required")
	}
	if c.FieldOfExpertise == "" {
		return status.Error(codes.InvalidArgument, "field of expertise is required")
	}
	return nil
}

// sagaTimeout must exceed the downstream participants' own timeouts, so that a
// stall is reported by the service that actually stalled rather than by every
// service in the chain at once.
const sagaTimeout = 20 * time.Second

type UpdateLecturerHandler struct {
	lecturerRepo  repo.ILecturerRepo
	db            *gorm.DB
	publisherConn *rabbitmq.PublisherConn
	sagaReplies   *saga.SagaReplyRegistry
}

func NewUpdateLecturerHandler(lecturerRepo repo.ILecturerRepo,
	db *gorm.DB,
	publisherConn *rabbitmq.PublisherConn,
	sagaReplies *saga.SagaReplyRegistry) *UpdateLecturerHandler {

	return &UpdateLecturerHandler{
		lecturerRepo:  lecturerRepo,
		db:            db,
		publisherConn: publisherConn,
		sagaReplies:   sagaReplies,
	}
}

func (h *UpdateLecturerHandler) Handle(ctx context.Context, cmd UpdateLecturerCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	lecturer, err := h.lecturerRepo.GetLecturerByID(ctx, cmd.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve lecturer")
	}
	if lecturer == nil {
		return status.Error(codes.NotFound, "lecturer not found")
	}

	lecturer.FullName = cmd.FullName
	lecturer.Title = cmd.Title
	lecturer.FieldOfExpertise = cmd.FieldOfExpertise

	// The saga runs inside the transaction: the local write only lands if every
	// downstream participant reports a commit, otherwise the tx rolls back.
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sagaID := uuid.New()
		ch := h.sagaReplies.Register(sagaID)
		defer h.sagaReplies.Unregister(sagaID)

		if err := h.publisherConn.PublishSaga(
			ctx,
			os.Getenv("RABBITMQ_LECTURER_TO_LECTURE_QUEUE"),
			sagaID,
			"UpdateLecturerSAGA",
			lecturer,
		); err != nil {
			log.Error().Err(err).Msgf("failed to dispatch saga %s for lecturer %d", sagaID, lecturer.Id)
			return status.Error(codes.Internal, "failed to dispatch lecturer saga")
		}

		select {
		case sagaErr := <-ch:
			if sagaErr != nil {
				log.Warn().Err(sagaErr).Msgf("saga %s rolled back for lecturer %d", sagaID, lecturer.Id)
				return status.Error(codes.Internal, "saga rolled back: "+sagaErr.Error())
			}
		case <-time.After(sagaTimeout):
			log.Warn().Msgf("saga %s timed out for lecturer %d", sagaID, lecturer.Id)
			return status.Error(codes.DeadlineExceeded, "saga reply timeout")
		}

		if err := h.lecturerRepo.WithTx(tx).UpdateLecturer(ctx, lecturer); err != nil {
			log.Error().Err(err).Msgf("failed to update lecturer %d after saga %s committed", lecturer.Id, sagaID)
			return status.Error(codes.Internal, "failed to update lecturer")
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
