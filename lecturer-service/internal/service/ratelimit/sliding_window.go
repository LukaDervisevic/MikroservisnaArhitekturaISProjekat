// Package ratelimit contains the pacing primitives used by the mail worker.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// SlidingWindow allows at most Limit events in any window of Period, based on
// the actual history of the events instead of a fixed delay between them. A
// burst may go through back to back until the window is full; from then on each
// caller waits exactly until the oldest event leaves the window.
type SlidingWindow struct {
	limit  int
	period time.Duration

	mutex sync.Mutex
	// events holds the timestamps still inside the current window, oldest first.
	events []time.Time

	// now and sleep are swapped out in tests to drive a virtual clock.
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

// NewSlidingWindow builds a limiter of limit events per period. A limit of zero
// or less disables throttling.
func NewSlidingWindow(limit int, period time.Duration) *SlidingWindow {
	return NewSlidingWindowWithClock(limit, period, time.Now, sleepCtx)
}

// NewSlidingWindowWithClock builds a limiter driven by an injected clock, so a
// test can exercise a one-minute budget without waiting a minute.
func NewSlidingWindowWithClock(
	limit int,
	period time.Duration,
	now func() time.Time,
	sleep func(ctx context.Context, d time.Duration) error,
) *SlidingWindow {
	return &SlidingWindow{
		limit:  limit,
		period: period,
		events: make([]time.Time, 0, max(limit, 1)),
		now:    now,
		sleep:  sleep,
	}
}

func (w *SlidingWindow) Limit() int { return w.limit }

func (w *SlidingWindow) Period() time.Duration { return w.period }

func (w *SlidingWindow) Wait(ctx context.Context) (time.Duration, error) {
	if w.limit <= 0 || w.period <= 0 {
		return 0, ctx.Err()
	}

	var waited time.Duration
	for {
		delay := w.reserve()
		if delay <= 0 {
			return waited, nil
		}
		if err := w.sleep(ctx, delay); err != nil {
			return waited, err
		}
		waited += delay
	}
}

func (w *SlidingWindow) reserve() time.Duration {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	now := w.now()
	cutoff := now.Add(-w.period)

	kept := w.events[:0]
	for _, event := range w.events {
		if event.After(cutoff) {
			kept = append(kept, event)
		}
	}
	w.events = kept

	if len(w.events) < w.limit {
		w.events = append(w.events, now)
		return 0
	}

	return w.events[0].Add(w.period).Sub(now)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
