package saga

import "sync"

type SagaReplyRegistry struct {
	mu    sync.Mutex
	chans map[int64]chan error
}

func NewSagaReplyRegistry() *SagaReplyRegistry {
	return &SagaReplyRegistry{chans: make(map[int64]chan error)}
}

func (r *SagaReplyRegistry) Register(id int64) chan error {
	ch := make(chan error, 1)
	r.mu.Lock()
	r.chans[id] = ch
	r.mu.Unlock()
	return ch
}

func (r *SagaReplyRegistry) Resolve(id int64, err error) {
	r.mu.Lock()
	ch, ok := r.chans[id]
	if ok {
		delete(r.chans, id)
	}
	r.mu.Unlock()
	if ok {
		ch <- err
	}
}
