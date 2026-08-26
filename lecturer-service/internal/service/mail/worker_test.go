package mail

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecturer-service/internal/service/ratelimit"
	"github.com/google/uuid"
)

const (
	testLimit  = 10
	testPeriod = time.Minute
	testBurst  = 25
)

// virtualClock replaces the wall clock, so a 10/min budget can be verified in
// milliseconds instead of minutes.
type virtualClock struct {
	mutex sync.Mutex
	now   time.Time
}

func (c *virtualClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *virtualClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = c.now.Add(d)
	return nil
}

// recordingSender stands in for the SMTP relay and remembers when each email
// was handed over, according to the virtual clock.
type recordingSender struct {
	clock *virtualClock
	sent  []model.EmailMessage
	times []time.Time
}

func (s *recordingSender) Send(_ context.Context, email model.EmailMessage) error {
	s.sent = append(s.sent, email)
	s.times = append(s.times, s.clock.Now())
	return nil
}

func (s *recordingSender) Describe() string { return "recording sender" }

// TestWorkerKeepsBurstWithinRateLimit dumps 25 emails on the worker at once and
// checks that no one-minute window ever contains more than 10 deliveries.
func TestWorkerKeepsBurstWithinRateLimit(t *testing.T) {
	clock := &virtualClock{now: time.Unix(0, 0).UTC()}
	sender := &recordingSender{clock: clock}
	limiter := ratelimit.NewSlidingWindowWithClock(testLimit, testPeriod, clock.Now, clock.Sleep)
	worker := NewWorker(limiter, sender)

	start := clock.Now()
	for i := 0; i < testBurst; i++ {
		email := model.EmailMessage{
			IdempotentKey: uuid.New(),
			To:            "lecturer@example.com",
			Subject:       "New lecture scheduled",
			Body:          "lecture number " + string(rune('a'+i%26)),
			EnqueuedAt:    start,
		}
		if err := worker.Process(context.Background(), email); err != nil {
			t.Fatalf("email %d failed: %v", i, err)
		}
	}

	if len(sender.sent) != testBurst {
		t.Fatalf("expected %d emails to be delivered, got %d", testBurst, len(sender.sent))
	}

	// The limit holds for *any* window, which for deliveries ordered in time
	// means the (i+limit)-th send must be at least one period after the i-th.
	for i := 0; i+testLimit < len(sender.times); i++ {
		gap := sender.times[i+testLimit].Sub(sender.times[i])
		if gap < testPeriod {
			t.Fatalf("emails %d..%d were sent within %s: %d sends in one %s window, limit is %d",
				i, i+testLimit, gap, testLimit+1, testPeriod, testLimit)
		}
	}

	// 25 emails at 10 per minute cannot be done in under two minutes.
	if elapsed := clock.Now().Sub(start); elapsed < 2*testPeriod {
		t.Fatalf("burst of %d finished in %s, which is faster than the limit allows", testBurst, elapsed)
	}
}

// TestWorkerFirstBatchIsNotThrottled documents that the limiter is dynamic: an
// empty history lets the first ten emails go out immediately.
func TestWorkerFirstBatchIsNotThrottled(t *testing.T) {
	clock := &virtualClock{now: time.Unix(0, 0).UTC()}
	worker := NewWorker(
		ratelimit.NewSlidingWindowWithClock(testLimit, testPeriod, clock.Now, clock.Sleep),
		&recordingSender{clock: clock},
	)

	for i := 0; i < testLimit; i++ {
		if err := worker.Process(context.Background(), model.EmailMessage{
			IdempotentKey: uuid.New(),
			To:            "lecturer@example.com",
			Subject:       "New lecture scheduled",
			EnqueuedAt:    clock.Now(),
		}); err != nil {
			t.Fatalf("email %d failed: %v", i, err)
		}
	}

	if elapsed := clock.Now().Sub(time.Unix(0, 0).UTC()); elapsed != 0 {
		t.Fatalf("the first %d emails should not be delayed, but took %s", testLimit, elapsed)
	}
}

// failingSender lets the retry/DLQ path be exercised without a relay.
type failingSender struct{ err error }

func (s *failingSender) Send(context.Context, model.EmailMessage) error { return s.err }
func (s *failingSender) Describe() string                               { return "failing sender" }

func TestWorkerPropagatesSendFailure(t *testing.T) {
	clock := &virtualClock{now: time.Unix(0, 0).UTC()}
	worker := NewWorker(
		ratelimit.NewSlidingWindowWithClock(testLimit, testPeriod, clock.Now, clock.Sleep),
		&failingSender{err: context.DeadlineExceeded},
	)

	err := worker.Process(context.Background(), model.EmailMessage{
		IdempotentKey: uuid.New(),
		To:            "lecturer@example.com",
	})
	if err == nil {
		t.Fatal("expected the delivery failure to reach the consumer so it can retry")
	}
}
