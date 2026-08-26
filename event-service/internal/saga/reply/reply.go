package reply

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
)

type Reply struct {
	Success      bool            `json:"success"`
	Error        string          `json:"error"`
	Compensation json.RawMessage `json:"compensation"`
	Output       json.RawMessage `json:"output"`
}

func (r Reply) Err() error {
	if r.Success {
		return nil
	}
	if r.Error == "" {
		return errors.New("participant rejected the step without a reason")
	}
	return errors.New(r.Error)
}

func Commit(compensation, output json.RawMessage) Reply {
	return Reply{Success: true, Compensation: compensation, Output: output}
}

func Fail(err error) Reply {
	return Reply{Success: false, Error: err.Error()}
}

type Registry struct {
	mu      sync.Mutex
	waiters map[uuid.UUID]chan Reply
}

func NewRegistry() *Registry {
	return &Registry{waiters: make(map[uuid.UUID]chan Reply)}
}

func (r *Registry) Register(correlationID uuid.UUID) chan Reply {
	ch := make(chan Reply, 1)
	r.mu.Lock()
	r.waiters[correlationID] = ch
	r.mu.Unlock()
	return ch
}

func (r *Registry) Unregister(correlationID uuid.UUID) {
	r.mu.Lock()
	delete(r.waiters, correlationID)
	r.mu.Unlock()
}

func (r *Registry) Resolve(correlationID uuid.UUID, rep Reply) bool {
	r.mu.Lock()
	ch, ok := r.waiters[correlationID]
	if ok {
		delete(r.waiters, correlationID)
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	ch <- rep
	return true
}
