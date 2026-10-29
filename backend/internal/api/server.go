package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"itassist/internal/storage"
	"itassist/internal/store"
)

type Authenticator func(r *http.Request) (store.User, error)

type Server struct {
	Store        *store.Store
	Files        *storage.Storage
	Auth         Authenticator
	Limit        *Limiter
	CookieSecure bool
	TrustProxy   bool
}

type userHandler func(w http.ResponseWriter, r *http.Request, u store.User)

func (s *Server) handle(mux *http.ServeMux, pattern, role string, h userHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		u, err := s.Auth(r)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if role != "" && u.Role != role {
			writeErr(w, http.StatusForbidden, "not available for your role")
			return
		}
		h(w, r, u)
	})
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)

	s.handle(mux, "GET /api/me", "", s.me)
	s.handle(mux, "GET /api/devices", "", s.devices)
	s.handle(mux, "GET /api/resolve-types", "", s.resolveTypes)

	s.handle(mux, "GET /api/tickets", "", s.listTickets)
	s.handle(mux, "POST /api/tickets", "employee", s.createTicket)
	s.handle(mux, "GET /api/tickets/{id}", "", s.getTicket)
	s.handle(mux, "GET /api/tickets/{id}/attachments/{aid}", "", s.getAttachment)
	s.handle(mux, "POST /api/tickets/{id}/messages", "", s.addMessage)
	s.handle(mux, "POST /api/tickets/{id}/accept", "support", s.acceptTicket)
	s.handle(mux, "POST /api/tickets/{id}/close", "support", s.closeTicket)

	s.handle(mux, "GET /api/notifications", "", s.listNotifications)
	s.handle(mux, "POST /api/notifications/read", "", s.markNotificationsRead)

	return recoverer(logger(mux))
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrForbidden):
		writeErr(w, http.StatusForbidden, err.Error())
	case errors.Is(err, store.ErrConflict), errors.Is(err, store.ErrLimit):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrInvalid):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("internal error: %v", err)
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic: %v", v)
				writeErr(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
