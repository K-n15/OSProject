package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"itassist/internal/store"
)

const (
	cookieName = "itassist_session"
	sessionTTL = 7 * 24 * time.Hour
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte(randomToken()), bcrypt.DefaultCost)
	return h
})

func SessionAuth(st *store.Store) Authenticator {
	return func(r *http.Request) (store.User, error) {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			return store.User{}, errors.New("no session")
		}
		return st.UserBySession(r.Context(), sha256Hex(c.Value))
	}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	username := strings.TrimSpace(in.Username)

	key := "login:" + s.clientIP(r) + ":" + strings.ToLower(username)
	if !s.Limit.Allow(key, 10, 15*time.Minute) || !s.Limit.Allow("login-ip:"+s.clientIP(r), 50, 15*time.Minute) {
		w.Header().Set("Retry-After", "900")
		writeErr(w, http.StatusTooManyRequests, "Too many attempts. Try again in a few minutes.")
		return
	}

	u, hash, err := s.Store.UserForLogin(r.Context(), username)
	if errors.Is(err, store.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(in.Password))
		writeErr(w, http.StatusUnauthorized, "Incorrect username or password")
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "Incorrect username or password")
		return
	}

	token := randomToken()
	if err := s.Store.CreateSession(r.Context(), u.ID, sha256Hex(token), sessionTTL); err != nil {
		fail(w, err)
		return
	}
	s.Limit.Reset(key)
	s.setSessionCookie(w, token, int(sessionTTL.Seconds()))
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		if err := s.Store.DeleteSession(r.Context(), sha256Hex(c.Value)); err != nil {
			log.Printf("logout: %v", err)
		}
	}
	s.setSessionCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
