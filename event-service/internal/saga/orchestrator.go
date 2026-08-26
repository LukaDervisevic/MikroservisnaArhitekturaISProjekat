package saga

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	sagarepo "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo/saga"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Definition struct {
	Name  string
	Steps []Step
}

type Orchestrator struct {
	store sagarepo.ISagaStore
}

func NewOrchestrator(store sagarepo.ISagaStore) *Orchestrator {
	return &Orchestrator{store: store}
}

type completedStep struct {
	index        int
	step         Step
	logID        uuid.UUID
	compensation json.RawMessage
}

func (o *Orchestrator) Run(ctx context.Context, def Definition, aggregateID int64, input any) (uuid.UUID, error) {
	sagaID := uuid.New()

	payload, err := json.Marshal(input)
	if err != nil {
		return sagaID, fmt.Errorf("marshal saga input: %w", err)
	}

	instance := &model.SagaInstance{
		ID:          sagaID,
		Name:        def.Name,
		AggregateID: aggregateID,
		State:       model.SagaRunning,
		CurrentStep: 0,
		Payload:     payload,
	}
	if err := o.store.CreateInstance(ctx, instance); err != nil {
		return sagaID, fmt.Errorf("persist saga instance: %w", err)
	}

	logger := log.With().
		Str("saga_id", sagaID.String()).
		Str("saga", def.Name).
		Int64("aggregate_id", aggregateID).
		Logger()
	logger.Info().Int("steps", len(def.Steps)).Msg("saga: started")

	sc := newContext(sagaID, input)
	completed := make([]completedStep, 0, len(def.Steps))

	for i, step := range def.Steps {
		stepLogID := uuid.New()
		stepLog := &model.SagaStepLog{
			ID:        stepLogID,
			SagaID:    sagaID,
			StepIndex: i,
			Name:      step.Name(),
			State:     model.StepExecuting,
		}
		if err := o.store.CreateStep(ctx, stepLog); err != nil {
			trackErr := fmt.Errorf("persist saga step %s: %w", step.Name(), err)
			logger.Error().Err(trackErr).Msg("saga: unable to record step, aborting")
			o.compensate(ctx, sagaID, def, sc, completed, trackErr)
			return sagaID, trackErr
		}

		_ = o.store.UpdateInstanceState(ctx, sagaID, model.SagaRunning, i, "")
		logger.Info().Int("step_index", i).Str("step", step.Name()).Msg("saga: executing step")

		result, err := step.Execute(ctx, sc)
		if err != nil {
			_ = o.store.UpdateStepState(ctx, stepLogID, model.StepFailed, nil, err.Error())
			logger.Error().Err(err).
				Int("step_index", i).
				Str("step", step.Name()).
				Msg("saga: step failed, starting compensation")

			o.compensate(ctx, sagaID, def, sc, completed, err)
			return sagaID, err
		}

		if err := o.store.UpdateStepState(ctx, stepLogID, model.StepCompleted, result.Compensation, ""); err != nil {
			logger.Error().Err(err).
				Int("step_index", i).
				Str("step", step.Name()).
				Msg("saga: step committed but its compensation could not be recorded")
		}

		sc.setOutput(step.Name(), result.Output)
		completed = append(completed, completedStep{
			index:        i,
			step:         step,
			logID:        stepLogID,
			compensation: result.Compensation,
		})

		logger.Info().Int("step_index", i).Str("step", step.Name()).Msg("saga: step committed")
	}

	if err := o.store.UpdateInstanceState(ctx, sagaID, model.SagaCompleted, len(def.Steps), ""); err != nil {
		logger.Error().Err(err).Msg("saga: completed but final state could not be recorded")
	}
	logger.Info().Msg("saga: completed, all steps committed")
	return sagaID, nil
}

func (o *Orchestrator) compensate(
	ctx context.Context,
	sagaID uuid.UUID,
	def Definition,
	sc *Context,
	completed []completedStep,
	cause error,
) {
	logger := log.With().
		Str("saga_id", sagaID.String()).
		Str("saga", def.Name).
		Logger()

	_ = o.store.UpdateInstanceState(ctx, sagaID, model.SagaCompensating, len(completed), cause.Error())

	if len(completed) == 0 {
		logger.Info().Msg("saga: nothing committed yet, no compensation needed")
		_ = o.store.UpdateInstanceState(ctx, sagaID, model.SagaAborted, 0, cause.Error())
		logger.Warn().Err(cause).Msg("saga: aborted")
		return
	}

	logger.Warn().
		Int("to_compensate", len(completed)).
		Err(cause).
		Msg("saga: compensating committed steps in reverse order")

	dirty := false
	for i := len(completed) - 1; i >= 0; i-- {
		c := completed[i]

		_ = o.store.UpdateStepState(ctx, c.logID, model.StepCompensating, nil, "")
		logger.Info().
			Int("step_index", c.index).
			Str("step", c.step.Name()).
			Msg("saga: compensating step")

		if err := c.step.Compensate(ctx, sc, c.compensation); err != nil {
			dirty = true
			_ = o.store.UpdateStepState(ctx, c.logID, model.StepCompensationFailed, nil, err.Error())
			logger.Error().Err(err).
				Int("step_index", c.index).
				Str("step", c.step.Name()).
				Msg("saga: COMPENSATION FAILED, participant left inconsistent")
			continue
		}

		_ = o.store.UpdateStepState(ctx, c.logID, model.StepCompensated, nil, "")
		logger.Info().
			Int("step_index", c.index).
			Str("step", c.step.Name()).
			Msg("saga: step compensated")
	}

	finalState := model.SagaAborted
	if dirty {
		finalState = model.SagaAbortedDirty
	}
	_ = o.store.UpdateInstanceState(ctx, sagaID, finalState, 0, cause.Error())
	logger.Warn().Err(cause).Str("state", string(finalState)).Msg("saga: aborted")
}
