package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Task struct {
	ID       uuid.UUID
	ExecFunc func() error
	Timeout  time.Duration
}

type Scheduler struct {
	queue   chan Task
	workers int
	stop    chan struct{}
}

func NewScheduler(workers int, queueCapacity int) *Scheduler {
	s := &Scheduler{
		queue:   make(chan Task, queueCapacity),
		workers: workers,
		stop:    make(chan struct{}),
	}
	s.Start()
	return s
}

func (s *Scheduler) Start() {
	for i := 0; i < s.workers; i++ {
		go func(workerID int) {
			for {
				select {
				case task := <-s.queue:
					// TODO: Check if context.Background() is appropriate
					ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
					defer cancel()
					done := make(chan error, 1)
					go func() {
						done <- task.ExecFunc()
					}()
					select {
					case err := <-done:
						if err != nil {
							log.Warn().Msgf("Worker %d failed task %s", i, task.ID.String())
						}
					case <-ctx.Done():
						log.Warn().Msgf("Worker %d timeout on task %s", i, task.ID.String())
					}
				case <-s.stop:
					return
				}
			}
		}(i)
	}
}

func (s *Scheduler) Submit(task Task) {
	s.queue <- task
}

func (s *Scheduler) Stop() {
	close(s.stop)
}
