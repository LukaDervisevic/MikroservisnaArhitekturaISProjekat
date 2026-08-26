package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga/reply"
)

const (
	StepUpdateEventRecord        = "UpdateEventRecord"
	StepUpdateEventProjection    = "UpdateEventProjection"
	StepUpdateEventReplica       = "UpdateEventReplica"
	StepUpdateLectureProjections = "UpdateLectureProjections"
)

const (
	MethodApplyEventProjection      = "SagaApplyEventProjection"
	MethodCompensateEventProjection = "SagaCompensateEventProjection"

	MethodApplyEventReplica      = "SagaApplyEventReplica"
	MethodCompensateEventReplica = "SagaCompensateEventReplica"

	MethodApplyLectureProjections      = "SagaApplyLectureProjections"
	MethodCompensateLectureProjections = "SagaCompensateLectureProjections"
)

const UpdateEventSagaName = "UpdateEventSaga"

const stepTimeout = 15 * time.Second

type UpdateEventInput struct {
	Event    *model.Event    `json:"event"`
	Location *model.Location `json:"location"`
}

type eventRecordCompensation struct {
	Event *model.Event `json:"event"`
}

type UpdateEventSagaDeps struct {
	EventRepo repo.IEventCommandRepo
	Publisher *rabbitmq.PublisherConn
	Replies   *reply.Registry
}

func NewUpdateEventSaga(deps UpdateEventSagaDeps) Definition {
	return Definition{
		Name: UpdateEventSagaName,
		Steps: []Step{
			&LocalStep{
				StepName: StepUpdateEventRecord,
				Do: func(ctx context.Context, sc *Context) (Result, error) {
					in := sc.Input.(UpdateEventInput)

					previous, err := deps.EventRepo.GetEventByID(ctx, in.Event.Id)
					if err != nil {
						return Result{}, fmt.Errorf("read event %d before update: %w", in.Event.Id, err)
					}
					if previous == nil {
						return Result{}, fmt.Errorf("event %d not found", in.Event.Id)
					}
					previous.Location = nil

					compensation, err := json.Marshal(eventRecordCompensation{Event: previous})
					if err != nil {
						return Result{}, fmt.Errorf("capture before-image of event %d: %w", in.Event.Id, err)
					}

					if err := deps.EventRepo.UpdateEvent(ctx, in.Event); err != nil {
						return Result{}, fmt.Errorf("update event %d: %w", in.Event.Id, err)
					}
					return Result{Compensation: compensation}, nil
				},
				Undo: func(ctx context.Context, sc *Context, compensation json.RawMessage) error {
					var c eventRecordCompensation
					if err := json.Unmarshal(compensation, &c); err != nil {
						return fmt.Errorf("decode event before-image: %w", err)
					}
					if c.Event == nil {
						return nil
					}
					c.Event.Location = nil
					return deps.EventRepo.UpdateEvent(ctx, c.Event)
				},
			},

			&RemoteStep{
				StepName:         StepUpdateEventProjection,
				Queue:            func() string { return os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE") },
				Method:           MethodApplyEventProjection,
				CompensateMethod: MethodCompensateEventProjection,
				Payload: func(sc *Context) (any, error) {
					in := sc.Input.(UpdateEventInput)
					return mapper.MapEventToQuery(in.Event, in.Location), nil
				},
				Publisher: deps.Publisher,
				Replies:   deps.Replies,
				Timeout:   stepTimeout,
			},

			&RemoteStep{
				StepName:         StepUpdateEventReplica,
				Queue:            func() string { return os.Getenv("RABBITMQ_EVENT_TO_LECTURE_QUEUE") },
				Method:           MethodApplyEventReplica,
				CompensateMethod: MethodCompensateEventReplica,
				Payload: func(sc *Context) (any, error) {
					in := sc.Input.(UpdateEventInput)
					return in.Event, nil
				},
				Publisher: deps.Publisher,
				Replies:   deps.Replies,
				Timeout:   stepTimeout,
			},

			&RemoteStep{
				StepName:         StepUpdateLectureProjections,
				Queue:            func() string { return os.Getenv("RABBITMQ_LECTURE_QUERY_QUEUE") },
				Method:           MethodApplyLectureProjections,
				CompensateMethod: MethodCompensateLectureProjections,
				Payload: func(sc *Context) (any, error) {
					projections, ok := sc.Output(StepUpdateEventReplica)
					if !ok {
						return nil, fmt.Errorf("step %s produced no lecture projections", StepUpdateEventReplica)
					}
					in := sc.Input.(UpdateEventInput)
					return applyLectureProjectionsPayload{
						EventID:     in.Event.Id,
						Projections: projections,
					}, nil
				},
				Publisher: deps.Publisher,
				Replies:   deps.Replies,
				Timeout:   stepTimeout,
			},
		},
	}
}

type applyLectureProjectionsPayload struct {
	EventID     int64           `json:"eventId"`
	Projections json.RawMessage `json:"projections"`
}
