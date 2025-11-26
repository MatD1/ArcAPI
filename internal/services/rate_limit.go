package services

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rateLimitBuffer        = 5
	rateLimitWaitPadding   = 100 * time.Millisecond
	rateLimitWaitRoundStep = 10 * time.Millisecond
)

type rateLimitGuard struct {
	mu        sync.Mutex
	seen      bool
	remaining int
	reset     time.Time
}

func (g *rateLimitGuard) wait(ctx context.Context) error {
	for {
		g.mu.Lock()
		shouldWait := g.shouldWaitLocked()
		reset := g.reset
		remaining := g.remaining
		g.mu.Unlock()

		if !shouldWait {
			return nil
		}

		waitDuration := time.Until(reset) + rateLimitWaitPadding
		if waitDuration <= 0 {
			continue
		}

		log.Printf("GitHub rate limit low (%d remaining). Pausing sync for %s", remaining, waitDuration.Round(rateLimitWaitRoundStep))
		select {
		case <-time.After(waitDuration):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (g *rateLimitGuard) shouldWaitLocked() bool {
	if !g.seen {
		return false
	}
	if g.remaining > rateLimitBuffer {
		return false
	}
	return g.reset.After(time.Now())
}

func (g *rateLimitGuard) updateFromHeaders(header http.Header) {
	var (
		remainingVal int
		resetVal     time.Time
		parsed       bool
	)

	if rem := strings.TrimSpace(header.Get("X-RateLimit-Remaining")); rem != "" {
		if parsedRem, err := strconv.Atoi(rem); err == nil {
			remainingVal = parsedRem
			if remainingVal < 0 {
				remainingVal = 0
			}
			parsed = true
		}
	}

	if reset := strings.TrimSpace(header.Get("X-RateLimit-Reset")); reset != "" {
		if parsedReset, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetVal = time.Unix(parsedReset, 0)
			parsed = true
		}
	}

	if !parsed {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if remainingVal >= 0 {
		g.remaining = remainingVal
		g.seen = true
	}
	if !resetVal.IsZero() {
		g.reset = resetVal
	}
}

type rateLimitAwareTransport struct {
	base  http.RoundTripper
	guard *rateLimitGuard
}

func (t *rateLimitAwareTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.guard.wait(req.Context()); err != nil {
		return nil, err
	}

	resp, err := t.base.RoundTrip(req)
	if resp != nil {
		t.guard.updateFromHeaders(resp.Header)
	}

	return resp, err
}

func newRateLimitAwareHTTPClient() *http.Client {
	return &http.Client{
		Transport: &rateLimitAwareTransport{
			base:  http.DefaultTransport,
			guard: &rateLimitGuard{},
		},
	}
}
