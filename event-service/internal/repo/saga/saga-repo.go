package saga

import (
	"context"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ISagaStore interface {
	CreateInstance(ctx context.Context, instance *model.SagaInstance) error
	UpdateInstanceState(ctx context.Context, sagaID uuid.UUID, state model.SagaState, currentStep int, lastErr string) error
	GetInstance(ctx context.Context, sagaID uuid.UUID) (*model.SagaInstance, error)

	CreateStep(ctx context.Context, step *model.SagaStepLog) error
	UpdateStepState(ctx context.Context, stepID uuid.UUID, state model.StepState, compensation []byte, lastErr string) error
	ListSteps(ctx context.Context, sagaID uuid.UUID) ([]model.SagaStepLog, error)
}

type SagaStore struct {
	db *gorm.DB
}

func NewSagaStore(db *gorm.DB) *SagaStore {
	return &SagaStore{db: db}
}

func (s *SagaStore) CreateInstance(ctx context.Context, instance *model.SagaInstance) error {
	now := time.Now().UTC()
	instance.CreatedAt = now
	instance.UpdatedAt = now
	return s.db.WithContext(ctx).Create(instance).Error
}

func (s *SagaStore) UpdateInstanceState(ctx context.Context, sagaID uuid.UUID, state model.SagaState, currentStep int, lastErr string) error {
	return s.db.WithContext(ctx).
		Model(&model.SagaInstance{}).
		Where("id = ?", sagaID).
		Updates(map[string]any{
			"state":        state,
			"current_step": currentStep,
			"last_error":   lastErr,
			"updated_at":   time.Now().UTC(),
		}).Error
}

func (s *SagaStore) GetInstance(ctx context.Context, sagaID uuid.UUID) (*model.SagaInstance, error) {
	var instance model.SagaInstance
	if err := s.db.WithContext(ctx).First(&instance, "id = ?", sagaID).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *SagaStore) CreateStep(ctx context.Context, step *model.SagaStepLog) error {
	step.StartedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Create(step).Error
}

func (s *SagaStore) UpdateStepState(ctx context.Context, stepID uuid.UUID, state model.StepState, compensation []byte, lastErr string) error {
	updates := map[string]any{
		"state": state,
	}
	if lastErr != "" {
		updates["last_error"] = lastErr
	}
	if compensation != nil {
		updates["compensation"] = compensation
	}
	switch state {
	case model.StepCompleted, model.StepFailed, model.StepCompensated, model.StepCompensationFailed:
		updates["finished_at"] = time.Now().UTC()
	}

	return s.db.WithContext(ctx).
		Model(&model.SagaStepLog{}).
		Where("id = ?", stepID).
		Updates(updates).Error
}

func (s *SagaStore) ListSteps(ctx context.Context, sagaID uuid.UUID) ([]model.SagaStepLog, error) {
	var steps []model.SagaStepLog
	err := s.db.WithContext(ctx).
		Where("saga_id = ?", sagaID).
		Order("step_index ASC").
		Find(&steps).Error
	return steps, err
}
