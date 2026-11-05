package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie   = "cork_session"
	sessionLifetime = 30 * 24 * time.Hour
	minPasswordLen  = 8
)

var usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]{2,32}$`)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

type userKey struct{}

func userFrom(r *http.Request) *User {
	u, _ := r.Context().Value(userKey{}).(*User)
	return u
}

func hashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

func (s *Server) currentUser(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	var u User
	var created, expires int64
	var isAdmin int
	err = s.db.QueryRow(`
		SELECT u.id, u.username, u.is_admin, u.created_at, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`,
		hashToken(c.Value), nowMS()).Scan(&u.ID, &u.Username, &isAdmin, &created, &expires)
	if err != nil {
		if !isNoRows(err) {
			slog.Error("session lookup", "err", err)
		}
		return nil
	}
	u.IsAdmin = isAdmin == 1
	u.CreatedAt = msTime(created)
	// sliding expiry: extend sessions that are past their half-life
	if time.Until(time.UnixMilli(expires)) < sessionLifetime/2 {
		s.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token_hash = ?`,
			time.Now().Add(sessionLifetime).UnixMilli(), hashToken(c.Value))
	}
	return &u
}

func (s *Server) requireUser(h func(http.ResponseWriter, *http.Request, *User)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r)
		if u == nil {
			writeError(w, http.StatusUnauthorized, "not signed in")
			return
		}
		h(w, r, u)
	})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID string) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	exp := time.Now().Add(sessionLifetime)
	if _, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		hashToken(tok), userID, nowMS(), exp.UnixMilli()); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    tok,
		Path:     "/",
		Expires:  exp.Add(sessionLifetime), // the server enforces real expiry
		HttpOnly: true,
		Secure:   s.isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Server) isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return s.cfg.TrustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
		if xr := r.Header.Get("X-Real-IP"); xr != "" {
			return xr
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- settings ---

func (s *Server) setting(key, def string) string {
	var v string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v); err != nil {
		return def
	}
	return v
}

func (s *Server) setSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Server) userCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Server) registrationOpen() bool { return s.setting("registration_open", "false") == "true" }

// --- user creation ---

var errUsernameTaken = errors.New("that username is taken")

func validateCredentials(username, password string) string {
	if !usernameRE.MatchString(username) {
		return "username must be 2–32 characters: letters, numbers, . _ or -"
	}
	if len(password) < minPasswordLen {
		return "password must be at least 8 characters"
	}
	if len(password) > 72 {
		return "password must be at most 72 characters"
	}
	return ""
}

func (s *Server) createUser(username, password string, isAdmin bool) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}
	u := &User{ID: newID(12), Username: username, IsAdmin: isAdmin, CreatedAt: msTime(nowMS())}
	_, err = s.db.Exec(`INSERT INTO users (id, username, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.ID, username, string(hash), boolInt(isAdmin), u.CreatedAt.UnixMilli())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, errUsernameTaken
		}
		return nil, err
	}
	return u, nil
}

func (s *Server) bootstrapAdmin() error {
	if s.cfg.AdminUsername == "" || s.cfg.AdminPassword == "" {
		return nil
	}
	n, err := s.userCount()
	if err != nil || n > 0 {
		return err
	}
	if msg := validateCredentials(s.cfg.AdminUsername, s.cfg.AdminPassword); msg != "" {
		return errors.New("CORK_ADMIN_USERNAME/CORK_ADMIN_PASSWORD: " + msg)
	}
	if _, err := s.createUser(s.cfg.AdminUsername, s.cfg.AdminPassword, true); err != nil {
		return err
	}
	slog.Info("created admin account from environment", "username", s.cfg.AdminUsername)
	return nil
}

// --- handlers ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	n, err := s.userCount()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"needs_setup":       n == 0,
		"registration_open": n == 0 || s.registrationOpen(),
		"user":              userFrom(r),
	})
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var registerMu sync.Mutex

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow(s.clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again in a few minutes")
		return
	}
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if msg := validateCredentials(c.Username, c.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// serialize so two simultaneous first sign-ups can't both become admin
	registerMu.Lock()
	n, err := s.userCount()
	if err != nil {
		registerMu.Unlock()
		serverError(w, err)
		return
	}
	first := n == 0
	if !first && !s.registrationOpen() {
		registerMu.Unlock()
		writeError(w, http.StatusForbidden, "registration is closed — ask your Cork admin for an account")
		return
	}
	u, err := s.createUser(c.Username, c.Password, first)
	registerMu.Unlock()
	if err != nil {
		if errors.Is(err, errUsernameTaken) {
			s.limiter.fail(s.clientIP(r))
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		serverError(w, err)
		return
	}
	if first {
		slog.Info("first user registered as admin", "username", u.Username)
	}
	if err := s.startSession(w, r, u.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// dummyHash makes failed logins for unknown users take as long as real ones.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("cork-timing-equalizer"), 12)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := s.clientIP(r)
	if !s.limiter.allow(ip) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again in a few minutes")
		return
	}
	var c credentials
	if !readJSON(w, r, &c) {
		return
	}
	var id, hash string
	err := s.db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, strings.TrimSpace(c.Username)).Scan(&id, &hash)
	if err != nil && !isNoRows(err) {
		serverError(w, err)
		return
	}
	if err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(c.Password))
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) != nil {
		s.limiter.fail(ip)
		writeError(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	if err := s.startSession(w, r, id); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, u *User) {
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request, u *User) {
	var body struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	var hash string
	if err := s.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, u.ID).Scan(&hash); err != nil {
		serverError(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Current)) != nil {
		writeError(w, http.StatusBadRequest, "current password is wrong")
		return
	}
	if msg := validateCredentials(u.Username, body.New); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.setPassword(u.ID, body.New); err != nil {
		serverError(w, err)
		return
	}
	// sign out every other session
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`, u.ID, hashToken(c.Value))
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) setPassword(userID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID)
	return err
}

// --- rate limiting (failed attempts per client IP) ---

type rateLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{max: max, window: window, hits: map[string][]time.Time{}}
}

func (l *rateLimiter) recent(key string) []time.Time {
	cut := time.Now().Add(-l.window)
	h := l.hits[key]
	i := 0
	for i < len(h) && h[i].Before(cut) {
		i++
	}
	h = h[i:]
	if len(h) == 0 {
		delete(l.hits, key)
	} else {
		l.hits[key] = h
	}
	return h
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key)) < l.max
}

func (l *rateLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits[key] = append(l.recent(key), time.Now())
}

func (l *rateLimiter) prune() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k := range l.hits {
		l.recent(k)
	}
}
