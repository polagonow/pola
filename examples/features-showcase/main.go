package main

import (
	"context"
	"log"
	"net/http"

	"github.com/polagonow/pola"
	"github.com/polagonow/pola/middleware/cors"
	"github.com/polagonow/pola/middleware/health"
)

func main() {
	// Showcase — two middleware borrowed from Goyave's ideas, registered as
	// plugins before the app boots:
	//   • CORS with preflight handling (middleware/cors)
	//   • liveness /healthz + readiness /readyz (middleware/health)
	pola.Use(
		cors.Plugin(
			cors.WithAllowedOrigins("*"),
			cors.WithAllowedMethods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete),
		),
		health.Plugin(
			health.WithCheck("app", func(_ context.Context) error { return nil }),
		),
	)

	if err := pola.Ready(); err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(pola.Addr(), nil))
}
