// A small HTTP service whose only job is to be deployed.
//
// It exists so the delivery pipeline has something real to carry: Jenkins
// builds it, pushes it to GHCR by digest, commits that digest to the GitOps
// repository, and Argo CD rolls it out. Every step of that is easier to trust
// when the thing moving through it can be read in one sitting.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Set at build time with -ldflags, so a running pod can say which commit it came
// from without anyone correlating timestamps.
var (
	version = "dev"
	commit  = "unknown"
)

type health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()

	// Readiness and liveness both point here for now because the service has no
	// dependencies. The moment it gains one — a database, a queue — readiness
	// should check it and liveness should not: a dependency being down means
	// this pod cannot serve, not that its process is wedged.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(health{Status: "ok", Version: version, Commit: commit})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": "demo-api",
			"version": version,
			"commit":  commit,
			"host":    hostname(),
		})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Kubernetes sends SIGTERM and then waits. Without this the process dies at
	// once and every request in flight becomes a 502 for whoever was making it.
	done := make(chan struct{})
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		logger.Info("shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
		close(done)
	}()

	logger.Info("listening", "addr", addr, "version", version, "commit", commit)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
	<-done
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
