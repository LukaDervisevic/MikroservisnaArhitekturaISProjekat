package rabbitmq

import (
	"encoding/json"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-query-service/internal/model"
)

type EventProjectionCompensation struct {
	EventID int64                    `json:"eventId"`
	Existed bool                     `json:"existed"`
	Row     *model.EventWithLocation `json:"row"`
}

const (
	MethodApplyEventProjection      = "SagaApplyEventProjection"
	MethodCompensateEventProjection = "SagaCompensateEventProjection"
	MethodRemoveEventProjection     = "SagaRemoveEventProjection"
)

type RemoveEventPayload struct {
	EventID int64 `json:"eventId"`
}

type SagaReply struct {
	Success      bool            `json:"success"`
	Error        string          `json:"error"`
	Compensation json.RawMessage `json:"compensation"`
	Output       json.RawMessage `json:"output"`
}

func CommitReply(compensation, output json.RawMessage) SagaReply {
	return SagaReply{Success: true, Compensation: compensation, Output: output}
}

func FailReply(err error) SagaReply {
	return SagaReply{Success: false, Error: err.Error()}
}
