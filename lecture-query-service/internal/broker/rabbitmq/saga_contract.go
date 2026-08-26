package rabbitmq

import (
	"encoding/json"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-query-service/internal/model"
)

const (
	MethodApplyLectureProjections      = "SagaApplyLectureProjections"
	MethodCompensateLectureProjections = "SagaCompensateLectureProjections"
)

type ApplyLectureProjectionsPayload struct {
	EventID     int64                `json:"eventId"`
	Projections []model.LectureQuery `json:"projections"`
}

type LectureProjectionsCompensation struct {
	EventID int64                `json:"eventId"`
	Rows    []model.LectureQuery `json:"rows"`
}

type SagaReply struct {
	Success      bool            `json:"success"`
	Error        string          `json:"error"`
	Compensation json.RawMessage `json:"compensation"`
	Output       json.RawMessage `json:"output"`
}

func CommitSagaReply(compensation, output json.RawMessage) SagaReply {
	return SagaReply{Success: true, Compensation: compensation, Output: output}
}

func FailSagaReply(err error) SagaReply {
	return SagaReply{Success: false, Error: err.Error()}
}
