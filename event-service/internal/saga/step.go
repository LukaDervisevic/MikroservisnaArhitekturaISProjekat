package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga/reply"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Context struct {
	SagaID uuid.UUID
	Input  any

	outputs map[string]json.RawMessage
}

func newContext(sagaID uuid.UUID, input any) *Context {
	return &Context{SagaID: sagaID, Input: input, outputs: make(map[string]json.RawMessage)}
}

func (c *Context) setOutput(step string, out json.RawMessage) {
	if len(out) > 0 {
		c.outputs[step] = out
	}
}

func (c *Context) Output(step string) (json.RawMessage, bool) {
	out, ok := c.outputs[step]
	return out, ok
}

// Result is what a step reports back to the orchestrator on success.
type Result struct {
	Compensation json.RawMessage
	Output       json.RawMessage
}

type Step interface {
	Name() string
	Execute(ctx context.Context, sc *Context) (Result, error)
	Compensate(ctx context.Context, sc *Context, compensation json.RawMessage) error
}

type LocalStep struct {
	StepName string
	Do       func(ctx context.Context, sc *Context) (Result, error)
	Undo     func(ctx context.Context, sc *Context, compensation json.RawMessage) error
}

func (s *LocalStep) Name() string { return s.StepName }

func (s *LocalStep) Execute(ctx context.Context, sc *Context) (Result, error) {
	return s.Do(ctx, sc)
}

func (s *LocalStep) Compensate(ctx context.Context, sc *Context, compensation json.RawMessage) error {
	if s.Undo == nil {
		return nil
	}
	return s.Undo(ctx, sc, compensation)
}

type RemoteStep struct {
	StepName         string
	Queue            func() string
	Method           string
	CompensateMethod string
	Payload          func(sc *Context) (any, error)

	Publisher *rabbitmq.PublisherConn
	Replies   *reply.Registry
	Timeout   time.Duration
}

func (s *RemoteStep) Name() string { return s.StepName }

func (s *RemoteStep) Execute(ctx context.Context, sc *Context) (Result, error) {
	body, err := s.Payload(sc)
	if err != nil {
		return Result{}, fmt.Errorf("build payload for %s: %w", s.StepName, err)
	}

	rep, err := s.dispatch(ctx, sc, s.Method, body)
	if err != nil {
		return Result{}, err
	}
	if err := rep.Err(); err != nil {
		return Result{}, err
	}
	return Result{Compensation: rep.Compensation, Output: rep.Output}, nil
}

func (s *RemoteStep) Compensate(ctx context.Context, sc *Context, compensation json.RawMessage) error {
	if s.CompensateMethod == "" {
		return nil
	}
	rep, err := s.dispatch(ctx, sc, s.CompensateMethod, json.RawMessage(compensation))
	if err != nil {
		return err
	}
	return rep.Err()
}

func (s *RemoteStep) dispatch(ctx context.Context, sc *Context, method string, body any) (reply.Reply, error) {
	correlationID := uuid.New()
	ch := s.Replies.Register(correlationID)
	defer s.Replies.Unregister(correlationID)

	queue := s.Queue()
	log.Info().
		Str("saga_id", sc.SagaID.String()).
		Str("step", s.StepName).
		Str("method", method).
		Str("queue", queue).
		Str("correlation_id", correlationID.String()).
		Msg("saga: dispatching command to participant")

	if err := s.Publisher.PublishSagaCommand(ctx, queue, sc.SagaID, correlationID, s.StepName, method, body); err != nil {
		return reply.Reply{}, fmt.Errorf("dispatch %s: %w", method, err)
	}

	select {
	case rep := <-ch:
		return rep, nil
	case <-time.After(s.Timeout):
		return reply.Reply{}, fmt.Errorf("timed out after %s waiting for %s reply", s.Timeout, method)
	case <-ctx.Done():
		return reply.Reply{}, ctx.Err()
	}
}
