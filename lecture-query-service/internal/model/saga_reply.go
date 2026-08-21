package model

import "errors"

type SagaReply struct {
	IsCommit bool   `json:"isCommit"`
	Reason   string `json:"reason"`
}

func (r SagaReply) Err() error {
	if r.IsCommit {
		return nil
	}
	if r.Reason == "" {
		return errors.New("saga rolled back by downstream service")
	}
	return errors.New(r.Reason)
}

func CommitReply() SagaReply {
	return SagaReply{IsCommit: true}
}

func RollbackReply(err error) SagaReply {
	return SagaReply{IsCommit: false, Reason: err.Error()}
}
