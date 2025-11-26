package services

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestRateLimitGuardWaitsUntilReset(t *testing.T) {
	guard := &rateLimitGuard{
		seen:      true,
		remaining: 0,
		reset:     time.Now().Add(50 * time.Millisecond),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	if err := guard.wait(ctx); err != nil {
		t.Fatalf("unexpected error waiting on guard: %v", err)
	}

	duration := time.Since(start)
	if duration < rateLimitWaitPadding {
		t.Fatalf("expected to wait at least padding (%s), waited %s", rateLimitWaitPadding, duration)
	}
}

func TestRateLimitGuardSkipsWhenPlentyRemaining(t *testing.T) {
	guard := &rateLimitGuard{
		seen:      true,
		remaining: rateLimitBuffer + 10,
		reset:     time.Now().Add(time.Minute),
	}

	ctx := context.Background()
	start := time.Now()
	if err := guard.wait(ctx); err != nil {
		t.Fatalf("unexpected error waiting on guard: %v", err)
	}

	if time.Since(start) > 10*time.Millisecond {
		t.Fatalf("expected immediate return but wait took %s", time.Since(start))
	}
}

func TestRateLimitGuardUpdatesFromHeaders(t *testing.T) {
	guard := &rateLimitGuard{}
	resetTime := time.Now().Add(time.Minute)
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "3")
	headers.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

	guard.updateFromHeaders(headers)

	guard.mu.Lock()
	defer guard.mu.Unlock()

	if guard.remaining != 3 {
		t.Fatalf("expected remaining to be 3, got %d", guard.remaining)
	}
	if guard.reset.Unix() != resetTime.Unix() {
		t.Fatalf("expected reset %d, got %d", resetTime.Unix(), guard.reset.Unix())
	}
}
