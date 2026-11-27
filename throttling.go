package rivex

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type ThrottleRate struct {
	Count   int
	Period  time.Duration
	Burst   int
}

func NewThrottleRate(count int, period time.Duration) ThrottleRate {
	return ThrottleRate{Count: count, Period: period, Burst: count}
}

func ParseThrottleRate(s string) (ThrottleRate, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ThrottleRate{}, fmt.Errorf("throttle: empty rate")
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return ThrottleRate{}, fmt.Errorf("throttle: invalid format: %s", s)
	}
	count, err := atoiSafe(parts[0])
	if err != nil {
		return ThrottleRate{}, fmt.Errorf("throttle: invalid count: %s", parts[0])
	}
	period, err := parseThrottlePeriod(parts[1])
	if err != nil {
		return ThrottleRate{}, fmt.Errorf("throttle: invalid period: %s", parts[1])
	}
	return ThrottleRate{Count: count, Period: period, Burst: count}, nil
}

func atoiSafe(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	if len(s) == 0 {
		return 0, fmt.Errorf("empty number")
	}
	return n, nil
}

func parseThrottlePeriod(s string) (time.Duration, error) {
	switch strings.ToLower(s) {
	case "sec", "second", "seconds", "s":
		return time.Second, nil
	case "min", "minute", "minutes", "m":
		return time.Minute, nil
	case "hour", "hours", "h":
		return time.Hour, nil
	case "day", "days", "d":
		return time.Hour * 24, nil
	}
	return time.ParseDuration(s)
}

type throttleBucket struct {
	timestamps []time.Time
}

func (b *throttleBucket) allow(rate ThrottleRate, now time.Time) (bool, time.Duration) {
	cutoff := now.Add(-rate.Period)
	valid := make([]time.Time, 0, len(b.timestamps))
	for _, ts := range b.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	b.timestamps = valid
	if len(b.timestamps) >= rate.Count {
		oldest := b.timestamps[0]
		wait := rate.Period - now.Sub(oldest)
		return false, wait
	}
	b.timestamps = append(b.timestamps, now)
	return true, 0
}

type Throttle struct {
	mu      sync.Mutex
	buckets map[string]*throttleBucket
	rate    ThrottleRate
}

func NewThrottle(rate ThrottleRate) *Throttle {
	return &Throttle{
		buckets: make(map[string]*throttleBucket),
		rate:    rate,
	}
}

func (t *Throttle) Allow(key string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	bucket, ok := t.buckets[key]
	if !ok {
		bucket = &throttleBucket{}
		t.buckets[key] = bucket
	}
	return bucket.allow(t.rate, time.Now())
}

func (t *Throttle) Reset(key string) {
	t.mu.Lock()
	delete(t.buckets, key)
	t.mu.Unlock()
}

type ThrottleKind int

const (
	ThrottleAnonymous ThrottleKind = iota
	ThrottleUser
	ThrottleScoped
)

type ThrottleOption func(*ThrottleScope)

type ThrottleScope struct {
	Kind   ThrottleKind
	Rate   ThrottleRate
	Scope  string
}

func ThrottleAnon(rate string) ThrottleOption {
	return func(t *ThrottleScope) {
		t.Kind = ThrottleAnonymous
		r, _ := ParseThrottleRate(rate)
		t.Rate = r
	}
}

func ThrottleUser_(rate string) ThrottleOption {
	return func(t *ThrottleScope) {
		t.Kind = ThrottleUser
		r, _ := ParseThrottleRate(rate)
		t.Rate = r
	}
}

func ThrottleScoped_(scope, rate string) ThrottleOption {
	return func(t *ThrottleScope) {
		t.Kind = ThrottleScoped
		t.Scope = scope
		r, _ := ParseThrottleRate(rate)
		t.Rate = r
	}
}

type ThrottlingMiddleware struct {
	throttles map[string]*Throttle
	mu        sync.RWMutex
}

func NewThrottlingMiddleware(opts ...ThrottleOption) *ThrottlingMiddleware {
	tm := &ThrottlingMiddleware{
		throttles: make(map[string]*Throttle),
	}
	for _, opt := range opts {
		scope := &ThrottleScope{}
		opt(scope)
		key := scope.Scope
		if key == "" {
			switch scope.Kind {
			case ThrottleAnonymous:
				key = "anon"
			case ThrottleUser:
				key = "user"
			}
		}
		tm.throttles[key] = NewThrottle(scope.Rate)
	}
	return tm
}

func (m *ThrottlingMiddleware) Process(ctx *Context, next NextFunc) error {
	key := m.resolveKey(ctx)
	for scope, throttle := range m.throttles {
		allowed, wait := throttle.Allow(key + ":" + scope)
		if !allowed {
			return NewHTTPException(StatusTooManyRequests, fmt.Sprintf("Throttled: %s. Retry after %v", scope, wait.Round(time.Second))).
				WithHeaders(map[string]string{"Retry-After": fmt.Sprintf("%d", int(wait.Seconds()))})
		}
	}
	return next(ctx)
}

func (m *ThrottlingMiddleware) resolveKey(ctx *Context) string {
	if ip := ctx.ClientHost(); ip != "" {
		return "ip:" + ip
	}
	return "anon"
}

type SlidingWindowRateLimiter struct {
	mu       sync.Mutex
	windows  map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windows: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (l *SlidingWindowRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	timestamps := l.windows[key]
	valid := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	if len(valid) >= l.limit {
		l.windows[key] = valid
		return false
	}
	valid = append(valid, now)
	l.windows[key] = valid
	return true
}

type TokenBucketLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucketState
	capacity int
	refill   int
	period   time.Duration
}

type tokenBucketState struct {
	tokens   int
	lastFill time.Time
}

func NewTokenBucketLimiter(capacity, refillPerPeriod int, period time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:  make(map[string]*tokenBucketState),
		capacity: capacity,
		refill:   refillPerPeriod,
		period:   period,
	}
}

func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	state, ok := l.buckets[key]
	if !ok {
		state = &tokenBucketState{tokens: l.capacity, lastFill: now}
		l.buckets[key] = state
	}
	elapsed := now.Sub(state.lastFill)
	refills := int(elapsed / l.period)
	if refills > 0 {
		state.tokens += refills * l.refill
		if state.tokens > l.capacity {
			state.tokens = l.capacity
		}
		state.lastFill = state.lastFill.Add(time.Duration(refills) * l.period)
	}
	if state.tokens <= 0 {
		return false
	}
	state.tokens--
	return true
}

type ThrottleStore interface {
	Get(key string) (*ThrottleState, bool)
	Set(key string, state *ThrottleState, ttl time.Duration)
}

type ThrottleState struct {
	Timestamps []time.Time
	mu         sync.Mutex
}

type InMemoryThrottleStore struct {
	data map[string]*storedThrottle
	mu   sync.Mutex
}

type storedThrottle struct {
	state     *ThrottleState
	expiresAt time.Time
}

func NewInMemoryThrottleStore() *InMemoryThrottleStore {
	return &InMemoryThrottleStore{data: make(map[string]*storedThrottle)}
}

func (s *InMemoryThrottleStore) Get(key string) (*ThrottleState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(stored.expiresAt) {
		delete(s.data, key)
		return nil, false
	}
	return stored.state, true
}

func (s *InMemoryThrottleStore) Set(key string, state *ThrottleState, ttl time.Duration) {
	s.mu.Lock()
	s.data[key] = &storedThrottle{state: state, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
}

func (s *InMemoryThrottleStore) Cleanup() {
	s.mu.Lock()
	now := time.Now()
	for k, v := range s.data {
		if now.After(v.expiresAt) {
			delete(s.data, k)
		}
	}
	s.mu.Unlock()
}
