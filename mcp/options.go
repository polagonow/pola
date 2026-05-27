package mcp

import (
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Transport selects how the MCP server speaks to clients.
type Transport string

const (
	// TransportHTTP uses the Streamable HTTP transport. Recommended for web apps.
	TransportHTTP Transport = "http"
	// TransportSSE uses the legacy Server-Sent Events transport.
	TransportSSE Transport = "sse"
	// TransportStdio reads JSON-RPC from stdin and writes to stdout. CLI use only.
	TransportStdio Transport = "stdio"
)

// Server is an alias for the underlying SDK server type, exposed so user
// plugins can DI-resolve it with core.Invoke[*mcp.Server].
type Server = *sdk.Server

// Option configures the MCP plugin.
type Option func(*config)

type config struct {
	transport    Transport
	mount        string
	name         string
	version      string
	instructions string
	sessionTTL   time.Duration
	recvMW       []sdk.Middleware
	sendMW       []sdk.Middleware
}

func defaultConfig() *config {
	return &config{
		transport: TransportHTTP,
		mount:     "/mcp",
		name:      "pola-mcp",
		version:   "0.1.0",
	}
}

// WithTransport sets the transport. Defaults to TransportHTTP.
func WithTransport(t Transport) Option {
	return func(c *config) { c.transport = t }
}

// WithMount sets the URL path prefix under which the MCP HTTP handler is
// served. Defaults to "/mcp". Ignored when Transport is stdio.
func WithMount(path string) Option {
	return func(c *config) { c.mount = path }
}

// WithName sets the server's Implementation.Name advertised to clients.
func WithName(name string) Option {
	return func(c *config) { c.name = name }
}

// WithVersion sets the server's Implementation.Version.
func WithVersion(v string) Option {
	return func(c *config) { c.version = v }
}

// WithInstructions sets the instructions string returned during the
// MCP initialize handshake.
func WithInstructions(s string) Option {
	return func(c *config) { c.instructions = s }
}

// WithSessionTTL sets the idle TTL for HTTP/SSE sessions. Zero means
// the SDK default.
func WithSessionTTL(d time.Duration) Option {
	return func(c *config) { c.sessionTTL = d }
}

// WithReceivingMiddleware attaches middleware that wraps incoming MCP
// requests (server-side).
func WithReceivingMiddleware(mw ...sdk.Middleware) Option {
	return func(c *config) { c.recvMW = append(c.recvMW, mw...) }
}

// WithSendingMiddleware attaches middleware that wraps outgoing MCP
// requests (server-to-client calls).
func WithSendingMiddleware(mw ...sdk.Middleware) Option {
	return func(c *config) { c.sendMW = append(c.sendMW, mw...) }
}
