package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	samberdo "github.com/samber/do/v2"

	pola "github.com/polagonow/pola"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/env"
	doinjection "github.com/polagonow/pola/injection/do"
	sloglogger "github.com/polagonow/pola/logger/slog"
	"github.com/polagonow/pola/middleware/logging"
	"github.com/polagonow/pola/middleware/recovery"
	"github.com/polagonow/pola/observability/metrics/prometheus"
	"github.com/polagonow/pola/observability/pprof"
	"github.com/polagonow/pola/observability/tracing/otel"

	// plugins register themselves via init()
	_ "github.com/polagonow/pola/bundler/esbuild" // requires: esbuild build tag
	_ "github.com/polagonow/pola/cache/memory"
	_ "github.com/polagonow/pola/engine/goja" // requires: goja build tag
	_ "github.com/polagonow/pola/fs/osfs"
	_ "github.com/polagonow/pola/logger/slog"
	_ "github.com/polagonow/pola/renderer/react" // requires: react build tag
	_ "github.com/polagonow/pola/router/nextjs"
)

func main() {
	e, err := env.Load()
	if err != nil {
		log.Fatalf("env: %v", err)
	}

	logger := sloglogger.New()

	// ── Dependency injection ───────────────────────────────────────────────
	injector := doinjection.New(samberdo.New())
	registerBlogServices(injector)

	// ── Registry ───────────────────────────────────────────────────────────
	reg := &core.Registry{
		Injectors: []core.RuntimeInjector{injector},
		Middleware: []core.Middleware{
			recovery.New(logger),
			logging.New(logger),
		},
	}
	if e.MetricsEnabled {
		reg.Metrics = prometheus.New()
	}
	if e.TracingEnabled {
		reg.Tracer = otel.New()
	}
	if e.PprofEnabled {
		reg.Pprof = pprof.New()
	}

	// ── Build app ──────────────────────────────────────────────────────────
	app, err := pola.New(&core.Config{
		WebAppPath: e.WebAppPath,
		Dev:        e.Dev,
		PublicDir:  e.PublicDir,
		Registry:   reg,
	})
	if err != nil {
		log.Fatalf("pola: %v", err)
	}

	// Build-only mode: generate assets to disk and exit (used by mage Bundle).
	if os.Getenv("POLA_BUILD_ONLY") == "true" {
		fmt.Println("pola: build complete")
		return
	}

	// ── HTTP server ────────────────────────────────────────────────────────
	addr := e.Address + ":" + e.Port

	mux := http.NewServeMux()
	mux.Handle("/", app)
	if e.MetricsEnabled {
		mux.Handle(e.MetricsPath, reg.Metrics.Handler())
	}
	if e.PprofEnabled {
		mux.Handle(e.PprofPath+"/", reg.Pprof.Handler())
	}

	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Pola listening on http://localhost%s\n", addr)

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

// registerBlogServices wires the blog-e2e demo data into the DI injector.
func registerBlogServices(inj *doinjection.Injector) {
	revisions := map[string][]map[string]any{
		"go-react-ssr": {
			{"rev": "v3", "date": "2024-01-15", "summary": "Published — added Suspense streaming section."},
			{"rev": "v2", "date": "2024-01-10", "summary": "Draft — expanded esbuild two-pass explanation."},
			{"rev": "v1", "date": "2024-01-05", "summary": "Initial draft — skeleton outline only."},
		},
		"rsc-deep-dive": {
			{"rev": "v2", "date": "2024-02-03", "summary": "Published — Flight wire-format diagrams added."},
			{"rev": "v1", "date": "2024-01-28", "summary": "Initial draft — protocol walkthrough."},
		},
		"goja-vm-internals": {
			{"rev": "v1", "date": "2024-03-10", "summary": "Published — first and only revision."},
		},
	}
	posts := []map[string]any{
		{"id": 1, "slug": "go-react-ssr", "title": "Building SSR with Go and React",
			"excerpt": "How to run React Server Components inside a Go process using Goja.",
			"author":  "Jane Doe", "date": "2024-01-15", "readTime": 5,
			"tags": []any{"go", "react", "ssr"}},
		{"id": 2, "slug": "rsc-deep-dive", "title": "React Server Components Deep Dive",
			"excerpt": "Understanding the Flight wire protocol and how RSC trees serialize.",
			"author":  "Jane Doe", "date": "2024-02-03", "readTime": 8,
			"tags": []any{"react", "rsc", "performance"}},
		{"id": 3, "slug": "goja-vm-internals", "title": "Goja VM Internals",
			"excerpt": "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
			"author":  "Jane Doe", "date": "2024-03-10", "readTime": 12,
			"tags": []any{"go", "javascript", "vm"}},
	}
	projects := []map[string]any{
		{"id": "1", "title": "GoJSX", "description": "Go-powered React SSR framework.",
			"tech": []any{"Go", "React", "TypeScript", "esbuild"}, "stars": 142, "status": "active"},
		{"id": "2", "title": "GojaBridge", "description": "Type-safe Go ↔ JS bridge.",
			"tech": []any{"Go", "Goja"}, "stars": 38, "status": "stable"},
		{"id": "3", "title": "FlightDecode", "description": "Pure-Go Flight wire-format decoder.",
			"tech": []any{"Go", "React"}, "stars": 21, "status": "beta"},
	}

	inj.
		Register("getPosts", func(_ []any) (any, error) { return posts, nil }).
		Register("getPost", func(args []any) (any, error) {
			slug := argStr(args, 0)

			fmt.Printf("Fetching post with slug %q...\n", slug)
			for _, p := range posts {
				if p["slug"] == slug {
					return p, nil
				}
			}
			return nil, fmt.Errorf("post %q not found", slug)
		}).
		Register("getProjects", func(_ []any) (any, error) {
			time.Sleep(1 * time.Second) // simulate latency
			return projects, nil
		}).
		Register("getProject", func(args []any) (any, error) {
			id := argStr(args, 0)
			for _, p := range projects {
				if p["id"] == id {
					return p, nil
				}
			}
			return nil, fmt.Errorf("project %q not found", id)
		}).
		Register("getRevisions", func(args []any) (any, error) {
			slug := argStr(args, 0)
			if revs, ok := revisions[slug]; ok {
				return revs, nil
			}
			return nil, fmt.Errorf("no revisions for post %q", slug)
		}).
		Register("getRevision", func(args []any) (any, error) {
			slug, rev := argStr(args, 0), argStr(args, 1)
			for _, r := range revisions[slug] {
				if r["rev"] == rev {
					return r, nil
				}
			}
			return nil, fmt.Errorf("revision %q not found for post %q", rev, slug)
		}).
		Register("getProfile", func(_ []any) (any, error) {
			return map[string]any{
				"id": "1", "name": "Jane Doe", "email": "jane@example.com",
				"role": "Senior Engineer", "bio": "Building dev tools.",
				"github": "janedoe", "website": "https://janedoe.dev",
			}, nil
		}).
		Register("triggerError", func(args []any) (any, error) {
			msg := "Forced error for testing"
			if s := argStr(args, 0); s != "" {
				msg = s
			}
			return nil, fmt.Errorf("%s", msg)
		})
}

func argStr(args []any, i int) string {
	if i < len(args) {
		return fmt.Sprintf("%v", args[i])
	}
	return ""
}
