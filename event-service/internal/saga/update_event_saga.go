package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/broker/rabbitmq"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	esservice "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/service"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/saga/reply"
	"gorm.io/gorm"
)

const (
	StepApplyEventChange         = "ApplyEventChange"
	StepUpdateEventProjection    = "UpdateEventProjection"
	StepUpdateEventReplica       = "UpdateEventReplica"
	StepUpdateLectureProjections = "UpdateLectureProjections"
)

const (
	MethodApplyEventProjection      = "SagaApplyEventProjection"
	MethodCompensateEventProjection = "SagaCompensateEventProjection"
	MethodRemoveEventProjection     = "SagaRemoveEventProjection"

	MethodApplyEventReplica      = "SagaApplyEventReplica"
	MethodCompensateEventReplica = "SagaCompensateEventReplica"
	MethodRemoveEventReplica     = "SagaRemoveEventReplica"

	MethodApplyLectureProjections      = "SagaApplyLectureProjections"
	MethodCompensateLectureProjections = "SagaCompensateLectureProjections"
	MethodRemoveLectureProjections     = "SagaRemoveLectureProjections"
)

const EventSagaName = "EventSaga"

const stepTimeout = 15 * time.Second

type EventOp string

const (
	OpCreate      EventOp = "create"
	OpUpdate      EventOp = "update"
	OpRename      EventOp = "rename"
	OpReschedule  EventOp = "reschedule"
	OpRelocate    EventOp = "relocate"
	OpChangePrice EventOp = "changePrice"
	OpCancel      EventOp = "cancel"
)

type EventChangeInput struct {
	EventID int64           `json:"eventId"`
	Op      EventOp         `json:"op"`
	Payload json.RawMessage `json:"payload"`
}

type EventFields struct {
	Name            string  `json:"name"`
	CotisationPrice float64 `json:"cotisationPrice"`
	Agenda          string  `json:"agenda"`
	Type            string  `json:"type"`
	DateTime        int64   `json:"dateTime"`
	LocationID      int64   `json:"locationId"`
}

func NewEventFieldsPayload(f EventFields) json.RawMessage { b, _ := json.Marshal(f); return b }
func NewNamePayload(n string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"newName": n})
	return b
}
func NewDateTimePayload(t int64) json.RawMessage {
	b, _ := json.Marshal(map[string]int64{"newDateTime": t})
	return b
}
func NewLocationPayload(id int64) json.RawMessage {
	b, _ := json.Marshal(map[string]int64{"newLocationId": id})
	return b
}
func NewPricePayload(p float64) json.RawMessage {
	b, _ := json.Marshal(map[string]float64{"newPrice": p})
	return b
}
func NewReasonPayload(r string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"reason": r})
	return b
}

type EventSagaDeps struct {
	Service   *esservice.EventAggregateService
	Locations repo.ILocationReadRepo
	Publisher *rabbitmq.PublisherConn
	Replies   *reply.Registry
}

type eventChangeCompensation struct {
	Before  aggregate.EventAggregateState `json:"before"`
	Created bool                          `json:"created"`
}

func input(sc *Context) EventChangeInput { return sc.Input.(EventChangeInput) }

func isCancel(sc *Context) bool { return input(sc).Op == OpCancel }

func NewEventSaga(deps EventSagaDeps) Definition {
	return Definition{
		Name: EventSagaName,
		Steps: []Step{
			applyEventChangeStep(deps),
			eventProjectionStep(deps),
			eventReplicaStep(deps),
			lectureProjectionsStep(deps),
		},
	}
}

func applyEventChangeStep(deps EventSagaDeps) Step {
	return &LocalStep{
		StepName: StepApplyEventChange,
		Do: func(ctx context.Context, sc *Context) (Result, error) {
			in := input(sc)
			var comp eventChangeCompensation
			comp.Created = in.Op == OpCreate

			err := deps.Service.DB().Transaction(func(tx *gorm.DB) error {
				agg, err := deps.Service.LoadTx(ctx, tx, in.EventID)
				if err != nil {
					return err
				}
				comp.Before = agg.ToState()
				if err := applyOp(agg, in); err != nil {
					return err
				}
				return deps.Service.AppendTx(ctx, tx, agg)
			})
			if err != nil {
				return Result{}, err
			}

			body, err := json.Marshal(comp)
			if err != nil {
				return Result{}, fmt.Errorf("marshal event change compensation: %w", err)
			}
			return Result{Compensation: body}, nil
		},
		Undo: func(ctx context.Context, sc *Context, compensation json.RawMessage) error {
			var c eventChangeCompensation
			if err := json.Unmarshal(compensation, &c); err != nil {
				return fmt.Errorf("decode event change compensation: %w", err)
			}
			return deps.Service.DB().Transaction(func(tx *gorm.DB) error {
				if c.Created {
					return deps.Service.DeleteAggregateTx(ctx, tx, c.Before.ID)
				}
				agg, err := deps.Service.LoadTx(ctx, tx, c.Before.ID)
				if err != nil {
					return err
				}
				if err := agg.RestoreTo(c.Before); err != nil {
					return err
				}
				return deps.Service.AppendTx(ctx, tx, agg)
			})
		},
	}
}

func applyOp(agg *aggregate.EventAggregate, in EventChangeInput) error {
	switch in.Op {
	case OpCreate:
		var f EventFields
		if err := json.Unmarshal(in.Payload, &f); err != nil {
			return err
		}
		return agg.Create(f.Name, f.CotisationPrice, f.Agenda, f.Type, f.DateTime, f.LocationID)
	case OpUpdate:
		var f EventFields
		if err := json.Unmarshal(in.Payload, &f); err != nil {
			return err
		}
		return agg.ApplyUpdate(f.Name, f.CotisationPrice, f.Agenda, f.Type, f.DateTime, f.LocationID)
	case OpRename:
		var p struct {
			NewName string `json:"newName"`
		}
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			return err
		}
		return agg.Rename(p.NewName)
	case OpReschedule:
		var p struct {
			NewDateTime int64 `json:"newDateTime"`
		}
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			return err
		}
		return agg.Reschedule(p.NewDateTime)
	case OpRelocate:
		var p struct {
			NewLocationID int64 `json:"newLocationId"`
		}
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			return err
		}
		return agg.Relocate(p.NewLocationID)
	case OpChangePrice:
		var p struct {
			NewPrice float64 `json:"newPrice"`
		}
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			return err
		}
		return agg.ChangePrice(p.NewPrice)
	case OpCancel:
		var p struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			return err
		}
		return agg.Cancel(p.Reason)
	default:
		return fmt.Errorf("unknown event op %q", in.Op)
	}
}

func (deps EventSagaDeps) resolveState(ctx context.Context, eventID int64) (aggregate.EventAggregateState, *model.Location, error) {
	agg, err := deps.Service.Load(ctx, eventID)
	if err != nil {
		return aggregate.EventAggregateState{}, nil, err
	}
	st := agg.ToState()
	loc, err := deps.Locations.GetLocationByID(ctx, st.LocationID)
	if err != nil {
		return st, nil, fmt.Errorf("resolve location %d: %w", st.LocationID, err)
	}
	if loc == nil {
		return st, nil, fmt.Errorf("location %d not found", st.LocationID)
	}
	return st, loc, nil
}

func stateToEvent(st aggregate.EventAggregateState) *model.Event {
	return &model.Event{
		Id:              st.ID,
		Name:            st.Name,
		CotisationPrice: st.CotisationPrice,
		Agenda:          st.Agenda,
		Type:            st.Type,
		DateTime:        st.DateTime,
		LocationID:      st.LocationID,
	}
}

func eventProjectionStep(deps EventSagaDeps) Step {
	return &RemoteStep{
		StepName:         StepUpdateEventProjection,
		Queue:            func() string { return os.Getenv("RABBITMQ_EVENT_QUERY_QUEUE") },
		Method:           func(sc *Context) string { return pick(sc, MethodApplyEventProjection, MethodRemoveEventProjection) },
		CompensateMethod: func(sc *Context) string { return MethodCompensateEventProjection },
		Payload: func(ctx context.Context, sc *Context) (any, error) {
			if isCancel(sc) {
				return map[string]int64{"eventId": input(sc).EventID}, nil
			}
			st, loc, err := deps.resolveState(ctx, input(sc).EventID)
			if err != nil {
				return nil, err
			}
			return mapper.MapEventToQuery(stateToEvent(st), loc), nil
		},
		Publisher: deps.Publisher,
		Replies:   deps.Replies,
		Timeout:   stepTimeout,
	}
}

func eventReplicaStep(deps EventSagaDeps) Step {
	return &RemoteStep{
		StepName:         StepUpdateEventReplica,
		Queue:            func() string { return os.Getenv("RABBITMQ_EVENT_TO_LECTURE_QUEUE") },
		Method:           func(sc *Context) string { return pick(sc, MethodApplyEventReplica, MethodRemoveEventReplica) },
		CompensateMethod: func(sc *Context) string { return MethodCompensateEventReplica },
		Payload: func(ctx context.Context, sc *Context) (any, error) {
			if isCancel(sc) {
				return map[string]int64{"eventId": input(sc).EventID}, nil
			}
			st, _, err := deps.resolveState(ctx, input(sc).EventID)
			if err != nil {
				return nil, err
			}
			return stateToEvent(st), nil
		},
		Publisher: deps.Publisher,
		Replies:   deps.Replies,
		Timeout:   stepTimeout,
	}
}

func lectureProjectionsStep(deps EventSagaDeps) Step {
	return &RemoteStep{
		StepName: StepUpdateLectureProjections,
		Queue:    func() string { return os.Getenv("RABBITMQ_LECTURE_QUERY_QUEUE") },
		Method: func(sc *Context) string {
			return pick(sc, MethodApplyLectureProjections, MethodRemoveLectureProjections)
		},
		CompensateMethod: func(sc *Context) string { return MethodCompensateLectureProjections },
		Payload: func(ctx context.Context, sc *Context) (any, error) {
			if isCancel(sc) {
				return map[string]int64{"eventId": input(sc).EventID}, nil
			}
			projections, ok := sc.Output(StepUpdateEventReplica)
			if !ok {
				return nil, fmt.Errorf("step %s produced no lecture projections", StepUpdateEventReplica)
			}
			return applyLectureProjectionsPayload{
				EventID:     input(sc).EventID,
				Projections: projections,
			}, nil
		},
		Publisher: deps.Publisher,
		Replies:   deps.Replies,
		Timeout:   stepTimeout,
	}
}

func pick(sc *Context, apply, remove string) string {
	if isCancel(sc) {
		return remove
	}
	return apply
}

type applyLectureProjectionsPayload struct {
	EventID     int64           `json:"eventId"`
	Projections json.RawMessage `json:"projections"`
}
