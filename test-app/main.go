package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pola "github.com/polagonow/pola"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/env"
)

func main() {
	e, err := env.Load()
	if err != nil {
		log.Fatalf("env: %v", err)
	}

	// ── Build app ──────────────────────────────────────────────────────────
	app, err := pola.New(
		core.WithWebAppPath(e.WebAppPath),
		core.WithPublicDir(e.PublicDir),
		core.WithDev(e.Dev),
	)
	if err != nil {
		log.Fatalf("pola: %v", err)
	}

	// Build-only mode: generate assets to disk and exit (used by pola build).
	if os.Getenv("POLA_BUILD_ONLY") == "true" {
		fmt.Println("pola: build complete")
		return
	}

	// ── HTTP server ────────────────────────────────────────────────────────
	addr := e.Address + ":" + e.Port

	mux := http.NewServeMux()
	mux.Handle("/", app)

	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Printf("test-app listening on http://localhost%s\n", addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
