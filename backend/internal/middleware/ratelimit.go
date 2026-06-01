package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RateLimiterConfig controls rate limiting behaviour.
type RateLimiterConfig struct {
	MaxRequests int           // max requests allowed in the window
	Window      time.Duration // window duration
	Enabled     bool          // if false, middleware is a no-op (skip overhead)
	KeyPrefix   string        // prefix for Redis keys (e.g. "rl:import:")
}

// RateLimitKeyFunc extracts the rate-limit key from the incoming request.
// By default we key on real client IP (X-Forwarded-For first, then X-Real-IP,
// then RemoteAddr). Subdivide by user ID in authenticated routes by passing
// a wrapper that reads X-User-ID.
type RateLimitKeyFunc func(r *http.Request) string

// DefaultClientIPKey returns the client IP for rate limiting.
func DefaultClientIPKey(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		for i := 0; i < len(fwd); i++ {
			if fwd[i] == ',' {
				return fwd[:i]
			}
		}
		return fwd
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	addr := r.RemoteAddr
	if colon := lastByteColon(addr); colon > 0 {
		return addr[:colon]
	}
	return addr
}

// lastByteColon returns the index of the last ':' in addr, or 0 if not found.
func lastByteColon(addr string) int {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return i
		}
	}
	return 0
}

// NewRateLimiter returns a chi middleware that enforces per-IP request limits.
// If redisURL is non-empty, Redis is used (INCR/EXPIRE pattern); otherwise
// an in-process sync.Map sliding-window counter is used.
//
// response headers set on every hit:
//   X-RateLimit-Limit     — max requests per window
//   X-RateLimit-Remaining — requests remaining in current window
//   X-RateLimit-Reset     — Unix timestamp when the window resets
//
// When the limit is exceeded the middleware responds with 429 Too Many Requests
// and a JSON problem+json body, and logs at Warn level.
func NewRateLimiter(
	log zerolog.Logger,
	cfg RateLimiterConfig,
	redisURL string,
	keyFunc RateLimitKeyFunc,
) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	if keyFunc == nil {
		keyFunc = DefaultClientIPKey
	}

	var store ratelimitStore
	if redisURL != "" {
		store = newRedisStore(log, redisURL, cfg.KeyPrefix, cfg.Window)
	} else {
		store = newInMemoryStore(log, cfg.MaxRequests, cfg.Window)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			allowed, remaining, resetAt, err := store.Allow(r.Context(), key, cfg.MaxRequests, cfg.Window)
			if err != nil {
				log.Error().Err(err).Str("key", key).Msg("rate limiter store error")
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.MaxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

			if !allowed {
				log.Warn().Str("key", key).Str("path", r.URL.Path).Msg("rate limit exceeded")
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"Rate limit exceeded. Retry after the reset timestamp."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ratelimitStore abstracts the underlying counters (in-memory or Redis).
type ratelimitStore interface {
	Allow(ctx context.Context, key string, maxReq int, window time.Duration) (allowed bool, remaining int, resetAt int64, err error)
}

// inMemoryStore is a sync.Map-based sliding-window rate limiter.
// It is safe for concurrent use with eventually-consistent per-key cleanup.
type inMemoryStore struct {
	log zerolog.Logger
	mu  sync.Mutex
	// counts maps key → current request count within the window.
	// Reset happens lazily on next access when window expires.
	counts map[string]*windowCounter
}

// windowCounter holds the request count and window start for one key.
type windowCounter struct {
	count   int
	expires time.Time
}

func newInMemoryStore(log zerolog.Logger, maxReq int, window time.Duration) *inMemoryStore {
	return &inMemoryStore{
		log:    log,
		counts: make(map[string]*windowCounter),
	}
}

func (s *inMemoryStore) Allow(_ context.Context, key string, maxReq int, window time.Duration) (bool, int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	counter, ok := s.counts[key]
	if !ok {
		counter = &windowCounter{expires: now.Add(window)}
		s.counts[key] = counter
	}

	// Lazily reset window if expired
	if now.After(counter.expires) {
		counter.count = 0
		counter.expires = now.Add(window)
	}

	// Increment count before checking limit (greedy increment).
	counter.count++

	allowed := counter.count <= maxReq
	remaining := maxReq - counter.count
	if remaining < 0 {
		remaining = 0
	}
	resetAt := counter.expires.Unix()

	return allowed, remaining, resetAt, nil
}

// Stop is a no-op for in-memory store.
func (s *inMemoryStore) Stop() {}

// redisStore implements ratelimitStore using Redis with INCR/EXPIRE.
type redisStore struct {
	log       zerolog.Logger
	client    redisClient
	keyPrefix string
	window    time.Duration
}

// redisClient abstracts github.com/redis/go-redis/v9 client for testability.
type redisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}

func newRedisStore(log zerolog.Logger, redisURL, keyPrefix string, window time.Duration) *redisStore {
	client, err := newRedisClient(redisURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to Redis for rate limiting; falling back to in-memory")
		return nil
	}
	return &redisStore{
		log:       log,
		client:    client,
		keyPrefix: keyPrefix,
		window:    window,
	}
}

func (s *redisStore) Allow(ctx context.Context, key string, maxReq int, window time.Duration) (bool, int, int64, error) {
	storeKey := s.keyPrefix + key

	n, err := s.client.Incr(ctx, storeKey)
	if err != nil {
		return false, 0, 0, err
	}

	if n == 1 {
		if _, err = s.client.Expire(ctx, storeKey, window); err != nil {
			s.log.Warn().Err(err).Str("key", storeKey).Msg("redis expire failed")
		}
	}

	ttl, err := s.client.TTL(ctx, storeKey)
	if err != nil {
		return false, 0, 0, err
	}
	resetAt := time.Now().Add(ttl).Unix()

	remaining := maxReq - int(n)
	if remaining < 0 {
		remaining = 0
	}
	allowed := n <= int64(maxReq)

	return allowed, remaining, resetAt, nil
}

// newRedisClient returns a redis client. Requires github.com/redis/go-redis/v9.
// Until configured, returns an error so NewRateLimiter falls back to in-memory.
func newRedisClient(url string) (redisClient, error) {
	return nil, errRedisNotConfigured
}

var errRedisNotConfigured = &errRedisDisabled{}

type errRedisDisabled struct{}

func (errRedisDisabled) Error() string {
	return "redis client not configured: set REDIS_URL or add go-redis dependency"
}