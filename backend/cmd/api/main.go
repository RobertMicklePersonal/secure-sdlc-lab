// Command api runs the notes HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/RobertMicklePersonal/secure-sdlc-lab/backend/internal/api"
	"github.com/RobertMicklePersonal/secure-sdlc-lab/backend/internal/notes"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	addr := envOr("ADDR", ":8080")
	maxNotes, err := strconv.Atoi(envOr("MAX_NOTES", "1000"))
	if err != nil || maxNotes < 1 {
		return errors.New("MAX_NOTES must be a positive integer")
	}

	handler := api.NewHandler(notes.NewStore(maxNotes), api.Config{
		AllowedOrigins: splitList(os.Getenv("ALLOWED_ORIGINS")),
		Logger:         log,
	})

	// Explicit timeouts and a header size cap keep slow or abusive clients
	// (Slowloris and similar) from tying up connections indefinitely.
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr, "max_notes", maxNotes)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
