package saga

import (
	"sync"

	"github.com/google/uuid"
)

type SagaReplyRegistry struct {
	mu    sync.Mutex
	chans map[uuid.UUID]chan error
}

func NewSagaReplyRegistry() *SagaReplyRegistry {
	return &SagaReplyRegistry{chans: make(map[uuid.UUID]chan error)}
}

func (r *SagaReplyRegistry) Register(sagaID uuid.UUID) chan error {
	ch := make(chan error, 1)
	r.mu.Lock()
	r.chans[sagaID] = ch
	r.mu.Unlock()
	return ch
}

func (r *SagaReplyRegistry) Unregister(sagaID uuid.UUID) {
	r.mu.Lock()
	delete(r.chans, sagaID)
	r.mu.Unlock()
}

func (r *SagaReplyRegistry) Resolve(sagaID uuid.UUID, err error) {
	r.mu.Lock()
	ch, ok := r.chans[sagaID]
	if ok {
		delete(r.chans, sagaID)
	}
	r.mu.Unlock()
	if ok {
		ch <- err
	}
}
