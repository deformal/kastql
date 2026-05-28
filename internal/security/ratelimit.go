package security

import (
	"net/http"
	"sync"
	"time"
)

// rateLimitState holds token buckets for global, per-IP, and mutation limits.
// It is recreated whenever RPM settings change.
type rateLimitState struct {
	globalBucket   *tokenBucket // nil when globalRPM == 0
	ipBuckets      sync.Map     // key: string IP → *tokenBucket
	mutationBucket *tokenBucket // nil when mutationRPM == 0
	perIPRPM       int
}

func newRateLimitState(globalRPM, perIPRPM, mutationRPM int) *rateLimitState {
	s := &rateLimitState{perIPRPM: perIPRPM}
	if globalRPM > 0 {
		s.globalBucket = newTokenBucket(globalRPM)
	}
	if mutationRPM > 0 {
		s.mutationBucket = newTokenBucket(mutationRPM)
	}
	return s
}

// AllowRequest returns true if the request is within global + per-IP limits.
func (s *rateLimitState) AllowRequest(ip string) bool {
	if s.globalBucket != nil && !s.globalBucket.take() {
		return false
	}
	if s.perIPRPM > 0 {
		bucket := s.bucketFor(ip)
		if !bucket.take() {
			return false
		}
	}
	return true
}

// AllowMutation returns true if mutation rate limit allows this request.
func (s *rateLimitState) AllowMutation() bool {
	if s.mutationBucket == nil {
		return true
	}
	return s.mutationBucket.take()
}

func (s *rateLimitState) bucketFor(ip string) *tokenBucket {
	if v, ok := s.ipBuckets.Load(ip); ok {
		return v.(*tokenBucket)
	}
	b := newTokenBucket(s.perIPRPM)
	actual, _ := s.ipBuckets.LoadOrStore(ip, b)
	return actual.(*tokenBucket)
}

// ── Token bucket ──────────────────────────────────────────────────────────────

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	refillPS float64 // tokens per second
	last     time.Time
}

func newTokenBucket(rpm int) *tokenBucket {
	rps := float64(rpm) / 60.0
	return &tokenBucket{
		tokens:   float64(rpm), // start full
		max:      float64(rpm),
		refillPS: rps,
		last:     time.Now(),
	}
}

func (b *tokenBucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	b.tokens += elapsed * b.refillPS
	if b.tokens > b.max {
		b.tokens = b.max
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ── HTTP middleware ───────────────────────────────────────────────────────────

// RateLimitMiddleware enforces global and per-IP request rate limits.
// Mutation rate checking is done inside graphqlHandler (after body parse).
func RateLimitMiddleware(mgr *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := mgr.Config()
			if !cfg.RateLimitEnabled {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			rl := mgr.RateLimiter()
			if !rl.AllowRequest(ip) {
				mgr.LogBlocked("rate_limit", ip, r.URL.Path)
				http.Error(w, `{"errors":[{"message":"rate limit exceeded"}]}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
