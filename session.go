package rivex

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"
)

const (
	sessionContextKey = "session"
	sessionCookieName = "session_id"
)

type SessionStore interface {
	Get(sessionID string) (*Session, bool)
	Set(sessionID string, session *Session)
	Delete(sessionID string)
	Cleanup(maxAge time.Duration)
}

type InMemorySessionStore struct {
	mu   sync.RWMutex
	data map[string]*Session
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		data: make(map[string]*Session),
	}
}

func (s *InMemorySessionStore) Get(sessionID string) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.data[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if sess.IsExpired() {
		s.Delete(sessionID)
		return nil, false
	}
	return sess, true
}

func (s *InMemorySessionStore) Set(sessionID string, session *Session) {
	s.mu.Lock()
	s.data[sessionID] = session
	s.mu.Unlock()
}

func (s *InMemorySessionStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.data, sessionID)
	s.mu.Unlock()
}

func (s *InMemorySessionStore) Cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, sess := range s.data {
		if now.Sub(sess.createdAt) > maxAge || sess.IsExpired() {
			delete(s.data, id)
		}
	}
}

type Session struct {
	ID        string
	data      map[string]interface{}
	mu        sync.RWMutex
	createdAt time.Time
	expiresAt time.Time
	dirty     bool
}

func NewSession(id string, ttl time.Duration) *Session {
	return &Session{
		ID:        id,
		data:      make(map[string]interface{}),
		createdAt: time.Now(),
		expiresAt: time.Now().Add(ttl),
		dirty:     true,
	}
}

func (s *Session) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	return v, ok
}

func (s *Session) GetString(key string) string {
	v, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, _ := v.(string)
	return str
}

func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	s.data[key] = value
	s.dirty = true
	s.mu.Unlock()
}

func (s *Session) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.dirty = true
	s.mu.Unlock()
}

func (s *Session) Clear() {
	s.mu.Lock()
	s.data = make(map[string]interface{})
	s.dirty = true
	s.mu.Unlock()
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s *Session) Touch(ttl time.Duration) {
	s.mu.Lock()
	s.expiresAt = time.Now().Add(ttl)
	s.dirty = true
	s.mu.Unlock()
}

func (s *Session) Data() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}

type SessionMiddleware struct {
	store    SessionStore
	name     string
	secret   []byte
	ttl      time.Duration
	secure   bool
	httpOnly bool
	sameSite string
}

type SessionOption func(*SessionMiddleware)

func WithSessionStore(store SessionStore) SessionOption {
	return func(s *SessionMiddleware) {
		s.store = store
	}
}

func WithSessionCookieName(name string) SessionOption {
	return func(s *SessionMiddleware) {
		s.name = name
	}
}

func WithSessionSecret(secret string) SessionOption {
	return func(s *SessionMiddleware) {
		if secret != "" {
			h := sha256.Sum256([]byte(secret))
			s.secret = h[:]
		}
	}
}

func WithSessionTTL(ttl time.Duration) SessionOption {
	return func(s *SessionMiddleware) {
		s.ttl = ttl
	}
}

func WithSessionSecure(secure bool) SessionOption {
	return func(s *SessionMiddleware) {
		s.secure = secure
	}
}

func WithSessionHTTPOnly(httpOnly bool) SessionOption {
	return func(s *SessionMiddleware) {
		s.httpOnly = httpOnly
	}
}

func WithSessionSameSite(sameSite string) SessionOption {
	return func(s *SessionMiddleware) {
		s.sameSite = sameSite
	}
}

func NewSessionMiddleware(opts ...SessionOption) *SessionMiddleware {
	sm := &SessionMiddleware{
		store:    NewInMemorySessionStore(),
		name:     sessionCookieName,
		ttl:      24 * time.Hour,
		secure:   false,
		httpOnly: true,
		sameSite: "Lax",
	}
	for _, opt := range opts {
		opt(sm)
	}
	if sm.secret == nil {
		key := make([]byte, 32)
		rand.Read(key)
		sm.secret = key
	}
	return sm
}

func (s *SessionMiddleware) Process(ctx *Context, next NextFunc) error {
	cookieValue := ctx.Cookie(s.name)
	var session *Session
	if cookieValue != "" {
		sessionID, err := s.decodeCookie(cookieValue)
		if err == nil {
			if existing, ok := s.store.Get(sessionID); ok {
				session = existing
			}
		}
	}
	if session == nil {
		sessionID := generateSessionID()
		session = NewSession(sessionID, s.ttl)
		s.store.Set(sessionID, session)
	}
	ctx.Set(sessionContextKey, session)
	err := next(ctx)
	if err != nil {
		return err
	}
	if session.dirty && ctx.response != nil {
		encoded := s.encodeCookie(session.ID)
		cookie := s.name + "=" + encoded +
			"; Path=/; Max-Age=" + intToStr(int(s.ttl.Seconds()))
		if s.httpOnly {
			cookie += "; HttpOnly"
		}
		if s.secure {
			cookie += "; Secure"
		}
		if s.sameSite != "" {
			cookie += "; SameSite=" + s.sameSite
		}
		ctx.response.Headers["Set-Cookie"] = cookie
		session.dirty = false
	}
	return nil
}

func (s *SessionMiddleware) decodeCookie(value string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(raw) < sha256.Size {
		return "", NewHTTPException(StatusBadRequest, "Invalid session cookie")
	}
	payload := raw[:len(raw)-sha256.Size]
	sig := raw[len(raw)-sha256.Size:]
	expected := sha256.Sum256(append(s.secret, payload...))
	if !hmacEqual(expected[:], sig) {
		return "", NewHTTPException(StatusBadRequest, "Invalid session cookie signature")
	}
	return string(payload), nil
}

func (s *SessionMiddleware) encodeCookie(sessionID string) string {
	payload := []byte(sessionID)
	sig := sha256.Sum256(append(s.secret, payload...))
	combined := append(payload, sig[:]...)
	return base64.RawURLEncoding.EncodeToString(combined)
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hexEncode(b)
}

func hexEncode(b []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexChars[v>>4]
		out[i*2+1] = hexChars[v&0x0f]
	}
	return string(out)
}

func GetSession(ctx *Context) *Session {
	v, ok := ctx.Get(sessionContextKey)
	if !ok {
		return nil
	}
	session, _ := v.(*Session)
	return session
}

func GetString(ctx *Context, key string) string {
	session := GetSession(ctx)
	if session == nil {
		return ""
	}
	return session.GetString(key)
}

func SessionToJSON(session *Session) ([]byte, error) {
	return json.Marshal(session.Data())
}
