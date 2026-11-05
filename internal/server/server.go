// Package server implements Cork's HTTP API, media serving and SPA hosting.
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tylxr59/cork/internal/scrape"
)

type Config struct {
	DataDir           string
	MaxUploadBytes    int64
	TrustProxy        bool
	AllowPrivateFetch bool
	// Optional: create this admin account on first start if no users exist.
	AdminUsername string
	AdminPassword string
	// Web is the built frontend (index.html + assets). May be nil in tests.
	Web fs.FS
}

type Server struct {
	cfg     Config
	db      *sql.DB
	fetch   *scrape.Client
	limiter *rateLimiter
	media   string
	handler http.Handler
}

func New(cfg Config) (*Server, error) {
	if cfg.MaxUploadBytes <= 0 {
		cfg.MaxUploadBytes = 25 << 20
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	media := filepath.Join(cfg.DataDir, "media")
	if err := os.MkdirAll(media, 0o755); err != nil {
		return nil, fmt.Errorf("data directory %s is not writable (in Docker, run `chown -R 65532:65532` on the host folder, or start the container with --user set to the folder's owner, e.g. --user 99:100 on Unraid): %w", cfg.DataDir, err)
	}
	db, err := openDB(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:     cfg,
		db:      db,
		fetch:   scrape.NewClient(cfg.AllowPrivateFetch),
		limiter: newRateLimiter(10, 5*time.Minute),
		media:   media,
	}
	if err := s.bootstrapAdmin(); err != nil {
		db.Close()
		return nil, err
	}
	s.handler = s.routes()
	go s.janitor()
	return s, nil
}

func (s *Server) Close() error { return s.db.Close() }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	api := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, h) }
	authed := func(pattern string, h func(http.ResponseWriter, *http.Request, *User)) {
		mux.Handle(pattern, s.requireUser(h))
	}
	admin := func(pattern string, h func(http.ResponseWriter, *http.Request, *User)) {
		mux.Handle(pattern, s.requireUser(func(w http.ResponseWriter, r *http.Request, u *User) {
			if !u.IsAdmin {
				writeError(w, http.StatusForbidden, "admin only")
				return
			}
			h(w, r, u)
		}))
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.PingContext(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	// auth
	api("GET /api/status", s.handleStatus)
	api("POST /api/auth/register", s.handleRegister)
	api("POST /api/auth/login", s.handleLogin)
	api("POST /api/auth/logout", s.handleLogout)
	authed("GET /api/me", s.handleMe)
	authed("PUT /api/me/password", s.handleChangePassword)

	// boards
	authed("GET /api/boards", s.handleListBoards)
	authed("POST /api/boards", s.handleCreateBoard)
	authed("PUT /api/boards/order", s.handleOrderBoards)
	api("GET /api/boards/{id}", s.handleGetBoard) // public boards are visible logged-out
	authed("PATCH /api/boards/{id}", s.handleUpdateBoard)
	authed("DELETE /api/boards/{id}", s.handleDeleteBoard)
	authed("PUT /api/boards/{id}/order", s.handleOrderPins)

	// pins
	authed("GET /api/pins", s.handleListPins)
	authed("POST /api/pins", s.handleCreatePin)
	api("GET /api/pins/{id}", s.handleGetPin)
	authed("PATCH /api/pins/{id}", s.handleUpdatePin)
	authed("PUT /api/pins/{id}/image", s.handleReplacePinImage)
	authed("DELETE /api/pins/{id}", s.handleDeletePin)
	authed("POST /api/scrape", s.handleScrape)

	// admin
	admin("GET /api/admin/users", s.handleAdminListUsers)
	admin("POST /api/admin/users", s.handleAdminCreateUser)
	admin("PATCH /api/admin/users/{id}", s.handleAdminUpdateUser)
	admin("DELETE /api/admin/users/{id}", s.handleAdminDeleteUser)
	admin("GET /api/admin/settings", s.handleAdminGetSettings)
	admin("PUT /api/admin/settings", s.handleAdminPutSettings)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	mux.HandleFunc("GET /media/{key}/{variant}", s.handleMedia)
	mux.Handle("/", s.spaHandler())

	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src * data: blob:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; "+
				"script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// CSRF: state-changing API calls must carry a custom header, which a
		// cross-site form or simple request can't set without a CORS preflight.
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("X-Cork") != "1" {
				writeError(w, http.StatusForbidden, "missing X-Cork header")
				return
			}
		}

		sw := &statusWriter{ResponseWriter: w, status: 200}
		user := s.currentUser(r)
		next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
		if strings.HasPrefix(r.URL.Path, "/api/") || sw.status >= 500 {
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status,
				"dur", time.Since(start).Round(time.Millisecond))
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// janitor periodically removes expired sessions.
func (s *Server) janitor() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if _, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, nowMS()); err != nil {
			slog.Warn("session cleanup failed", "err", err)
		}
		s.limiter.prune()
		<-t.C
	}
}

// --- helpers ---

type apiError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
}

func serverError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)
	writeError(w, http.StatusInternalServerError, "something went wrong")
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
