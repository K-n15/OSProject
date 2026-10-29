package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"itassist/internal/api"
	"itassist/internal/config"
	"itassist/internal/storage"
	"itassist/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err) // a malformed DATABASE_URL; retrying won't help
	}
	defer pool.Close()
	if err := retry(ctx, "database", func() error { return pool.Ping(ctx) }); err != nil {
		log.Fatal(err)
	}

	files, err := storage.New(ctx, cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := retry(ctx, "storage", func() error { return files.EnsureBucket(ctx) }); err != nil {
		log.Fatal(err)
	}

	if !cfg.CookieSecure {
		log.Printf("COOKIE_SECURE=0: session cookies work over plain HTTP. Use this for local development only.")
	}

	st := store.New(pool)
	srv := &api.Server{
		Store:        st,
		Files:        files,
		Auth:         api.SessionAuth(st),
		Limit:        api.NewLimiter(),
		CookieSecure: cfg.CookieSecure,
		TrustProxy:   cfg.TrustProxy,
	}

	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			if err := st.DeleteExpiredSessions(ctx); err != nil && ctx.Err() == nil {
				log.Printf("cleanup sessions: %v", err)
			}
			if err := st.DeleteOldNotifications(ctx); err != nil && ctx.Err() == nil {
				log.Printf("cleanup notifications: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("IT-Assist API listening on %s", cfg.Addr)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func retry(ctx context.Context, what string, fn func() error) error {
	wait := time.Second
	deadline := time.Now().Add(60 * time.Second)
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s still unavailable after 60s: %w", what, err)
		}
		log.Printf("waiting for %s (%v)", what, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		if wait < 8*time.Second {
			wait *= 2
		}
	}
}
