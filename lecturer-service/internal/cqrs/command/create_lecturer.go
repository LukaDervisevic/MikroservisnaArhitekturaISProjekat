package command

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/repo"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type CreateLecturerCommand struct {
	FullName         string
	Title            string
	FieldOfExpertise string
}

type CreateLecturerHandler struct {
	db           *gorm.DB
	lecturerRepo *repo.LecturerRepo
	outboxRepo   *repo.OutboxRepo
}

func NewCreateLecturerHandler(db *gorm.DB, lecturerRepo *repo.LecturerRepo, outboxRepo *repo.OutboxRepo) *CreateLecturerHandler {
	return &CreateLecturerHandler{db: db, lecturerRepo: lecturerRepo, outboxRepo: outboxRepo}
}

func (h *CreateLecturerHandler) Handle(ctx context.Context, cmd CreateLecturerCommand) (*model.Lecturer, error) {

	lecturer := &model.Lecturer{
		FullName:         cmd.FullName,
		Title:            cmd.Title,
		FieldOfExpertise: cmd.FieldOfExpertise,
	}

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := h.lecturerRepo.WithTx(tx).CreateLecturer(ctx, lecturer); err != nil {
			return err
		}

		outboxMsg := rabbitmq.Message[interface{}]{
			IdempotentKey: uuid.New(),
			Body:          *lecturer,
			Method:        "CreateLecturer",
		}

		if err := h.outboxRepo.WithTx(tx).StashMessage(ctx, outboxMsg); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to create lecturer and outbox event")
		return nil, status.Error(codes.Internal, "failed to create lecturer")
	}

	return lecturer, nil
}
