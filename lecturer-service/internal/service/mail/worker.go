package mail

import (
	"context"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/rs/zerolog/log"
)

type Sender interface {
	Send(ctx context.Context, email model.EmailMessage) error
	Describe() string
}

type Limiter interface {
	Wait(ctx context.Context) (time.Duration, error)
	Limit() int
	Period() time.Duration
}

type Worker struct {
	limiter Limiter
	sender  Sender
}

func NewWorker(limiter Limiter, sender Sender) *Worker {
	return &Worker{limiter: limiter, sender: sender}
}

func (w *Worker) Process(ctx context.Context, email model.EmailMessage) error {
	log.Info().
		Str("to", email.To).
		Str("id", email.IdempotentKey.String()).
		Int("attempt", email.RetryCount+1).
		Msg("mail worker started processing email")

	waited, err := w.limiter.Wait(ctx)
	if err != nil {
		return err
	}
	if waited > 0 {
		log.Warn().
			Str("to", email.To).
			Str("id", email.IdempotentKey.String()).
			Dur("waited", waited).
			Int("limit", w.limiter.Limit()).
			Dur("period", w.limiter.Period()).
			Msg("mail worker throttled by rate limit")
	}

	if err := w.sender.Send(ctx, email); err != nil {
		return err
	}

	log.Info().
		Str("to", email.To).
		Str("id", email.IdempotentKey.String()).
		Str("transport", w.sender.Describe()).
		Dur("queue_latency", time.Since(email.EnqueuedAt)).
		Msg("email sent")
	return nil
}
