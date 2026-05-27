package mcp

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildServer constructs an *sdk.Server and applies package-level
// init()-time registrations to it. User plugins add their tools later
// by resolving *sdk.Server from the DI container in their own Start hook.
func buildServer(cfg *config) *sdk.Server {
	srv := sdk.NewServer(
		&sdk.Implementation{Name: cfg.name, Version: cfg.version},
		&sdk.ServerOptions{Instructions: cfg.instructions},
	)

	tools, res, resTpl, prompts, typed := snapshot()
	for _, t := range tools {
		srv.AddTool(t.Tool, t.Handler)
	}
	for _, e := range res {
		srv.AddResource(e.Resource, e.Handler)
	}
	for _, e := range resTpl {
		srv.AddResourceTemplate(e.Template, e.Handler)
	}
	for _, e := range prompts {
		srv.AddPrompt(e.Prompt, e.Handler)
	}
	for _, fn := range typed {
		fn(srv)
	}

	if len(cfg.recvMW) > 0 {
		srv.AddReceivingMiddleware(cfg.recvMW...)
	}
	if len(cfg.sendMW) > 0 {
		srv.AddSendingMiddleware(cfg.sendMW...)
	}

	return srv
}

// buildHTTPHandler returns an http.Handler for the chosen transport, or
// nil if the transport doesn't speak HTTP (e.g. stdio).
func buildHTTPHandler(srv *sdk.Server, cfg *config) http.Handler {
	getServer := func(_ *http.Request) *sdk.Server { return srv }
	switch cfg.transport {
	case TransportHTTP:
		return sdk.NewStreamableHTTPHandler(getServer, nil)
	case TransportSSE:
		return sdk.NewSSEHandler(getServer, nil)
	}
	return nil
}
