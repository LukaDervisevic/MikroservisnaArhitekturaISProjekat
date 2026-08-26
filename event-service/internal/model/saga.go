package model

import (
	"time"

	"github.com/google/uuid"
)

type SagaState string

const (
	SagaRunning      SagaState = "RUNNING"
	SagaCompensating SagaState = "COMPENSATING"
	SagaCompleted    SagaState = "COMPLETED"
	SagaAborted      SagaState = "ABORTED"
	SagaAbortedDirty SagaState = "ABORTED_DIRTY"
)

type StepState string

const (
	StepExecuting          StepState = "EXECUTING"
	StepCompleted          StepState = "COMPLETED"
	StepFailed             StepState = "FAILED"
	StepCompensating       StepState = "COMPENSATING"
	StepCompensated        StepState = "COMPENSATED"
	StepCompensationFailed StepState = "COMPENSATION_FAILED"
)

type SagaInstance struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Name        string    `gorm:"column:name"`
	AggregateID int64     `gorm:"column:aggregate_id"`
	State       SagaState `gorm:"column:state"`
	CurrentStep int       `gorm:"column:current_step"`
	Payload     []byte    `gorm:"column:payload"`
	LastError   string    `gorm:"column:last_error"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type SagaStepLog struct {
	ID           uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	SagaID       uuid.UUID  `gorm:"column:saga_id;type:uuid"`
	StepIndex    int        `gorm:"column:step_index"`
	Name         string     `gorm:"column:name"`
	State        StepState  `gorm:"column:state"`
	Compensation []byte     `gorm:"column:compensation"`
	LastError    string     `gorm:"column:last_error"`
	StartedAt    time.Time  `gorm:"column:started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at"`
}
