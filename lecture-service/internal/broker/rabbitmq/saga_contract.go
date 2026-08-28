package rabbitmq

import (
	"encoding/json"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
)

const (
	MethodApplyEventReplica      = "SagaApplyEventReplica"
	MethodCompensateEventReplica = "SagaCompensateEventReplica"
	MethodRemoveEventReplica     = "SagaRemoveEventReplica"
)

type RemoveEventPayload struct {
	EventID int64 `json:"eventId"`
}

type EventReplicaCompensation struct {
	EventID int64        `json:"eventId"`
	Existed bool         `json:"existed"`
	Row     *model.Event `json:"row"`
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
