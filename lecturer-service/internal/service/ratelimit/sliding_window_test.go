package ratelimit

import (
	"context"
	"testing"
	"time"
)

// fakeClock lets the tests advance time instead of waiting for it.
type fakeClock struct {
	now time.Time
}

func (c *fakeClock) window(limit int, period time.Duration) *SlidingWindow {
	return NewSlidingWindowWithClock(limit, period,
		func() time.Time { return c.now },
		func(ctx context.Context, d time.Duration) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			c.now = c.now.Add(d)
			return nil
		})
}

func TestBurstUpToLimitIsNotDelayed(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	window := clock.window(10, time.Minute)

	for i := 0; i < 10; i++ {
		waited, err := window.Wait(context.Background())
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if waited != 0 {
			t.Fatalf("call %d: expected no delay inside the window, waited %s", i, waited)
		}
	}
}

func TestCallOverLimitWaitsForOldestToLeaveWindow(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	window := clock.window(10, time.Minute)

	for i := 0; i < 10; i++ {
		if _, err := window.Wait(context.Background()); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}

	// Half a minute in, the window is still full: the 11th call has to wait out
	// the remaining 30s, not a full period and not a fixed delay.
	clock.now = clock.now.Add(30 * time.Second)
	waited, err := window.Wait(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if waited != 30*time.Second {
		t.Fatalf("expected to wait the remainder of the window (30s), waited %s", waited)
	}
}

func TestHistoryDrivesPacingNotAFixedDelay(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	window := clock.window(10, time.Minute)

	// Two calls now, then a long idle gap: the idle time frees the whole window,
	// so the next burst of ten must go through untouched.
	for i := 0; i < 2; i++ {
		if _, err := window.Wait(context.Background()); err != nil {
			t.Fatalf("warm-up call %d: %v", i, err)
		}
	}
	clock.now = clock.now.Add(2 * time.Minute)

	for i := 0; i < 10; i++ {
		waited, err := window.Wait(context.Background())
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if waited != 0 {
			t.Fatalf("call %d after an idle period should not be delayed, waited %s", i, waited)
		}
	}
}

func TestWaitStopsOnCancelledContext(t *testing.T) {
	window := NewSlidingWindow(1, time.Minute)
	if _, err := window.Wait(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := window.Wait(ctx); err == nil {
		t.Fatal("expected the cancelled context to abort the wait")
	}
}
