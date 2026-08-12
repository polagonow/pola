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
//	import { Server } from "@pola/actions"
//	const info = await Server.getServerInfo()
type Server struct{}

type ServerInfo struct {
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Time      string `json:"time"`
}

type Greeting struct {
	Message string `json:"message"`
}

// GetServerInfo returns server metadata. Becomes Server.getServerInfo() in JS.
func (s *Server) GetServerInfo() (*ServerInfo, error) {
	return &ServerInfo{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Time:      time.Now().Format(time.RFC3339),
	}, nil
}

// Greet returns a greeting message. Becomes Server.greet(name) in JS.
func (s *Server) Greet(name string) (*Greeting, error) {
	if name == "" {
		name = "World"
	}
	return &Greeting{
		Message: fmt.Sprintf("Hello, %s! This message was generated in Go.", name),
	}, nil
}
