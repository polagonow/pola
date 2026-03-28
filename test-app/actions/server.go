package actions

import (
	"fmt"
	"runtime"
	"time"
)

// Server exposes server metadata and utilities to JavaScript via the Pola bridge.
//
// Each exported method becomes a callable function in JS.
// Method names are automatically camelCased: GetServerInfo → Server.getServerInfo()
//
// Usage in React:
//
//	import { Server } from "@pola/di"
//	const info = await Server.getServerInfo()
type Server struct{}

// GetServerInfo returns server metadata. Becomes Server.getServerInfo() in JS.
func (s *Server) GetServerInfo() (map[string]any, error) {
	return map[string]any{
		"goVersion": runtime.Version(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"time":      time.Now().Format(time.RFC3339),
	}, nil
}

// Greet returns a greeting message. Becomes Server.greet(name) in JS.
func (s *Server) Greet(name string) (map[string]any, error) {
	if name == "" {
		name = "World"
	}
	return map[string]any{
		"message": fmt.Sprintf("Hello, %s! This message was generated in Go.", name),
	}, nil
}
