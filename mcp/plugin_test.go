package mcp

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
)

func TestPluginName(t *testing.T) {
	if got := Plugin().Name(); got != "mcp" {
		t.Fatalf("Plugin().Name() = %q, want %q", got, "mcp")
	}
}

func TestPluginRegistersServerInDI(t *testing.T) {
	resetForTests()

	reg := core.NewRegistry()
	p := Plugin(WithMount("/mcp"), WithName("test"), WithVersion("0.0.0"))
	p.Register(reg)

	srv, err := core.Invoke[*sdk.Server](reg)
	if err != nil {
		t.Fatalf("Invoke[*sdk.Server]: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil *sdk.Server in DI container")
	}
}

func TestHTTPTransportAddsMiddleware(t *testing.T) {
	resetForTests()

	reg := core.NewRegistry()
	p := Plugin(WithTransport(TransportHTTP), WithMount("/mcp"))
	p.Register(reg)

	mws := reg.Middleware()
	found := false
	for _, m := range mws {
		if m.Name() == "mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mcp middleware to be registered; got %d middlewares", len(mws))
	}
}

func TestStdioTransportSkipsMiddleware(t *testing.T) {
	resetForTests()

	reg := core.NewRegistry()
	p := Plugin(WithTransport(TransportStdio))
	p.Register(reg)

	for _, m := range reg.Middleware() {
		if m.Name() == "mcp" {
			t.Fatalf("stdio transport should not register HTTP middleware")
		}
	}
}

func TestStartReturnsNilForHTTP(t *testing.T) {
	resetForTests()

	reg := core.NewRegistry()
	p := Plugin(WithTransport(TransportHTTP))
	p.Register(reg)

	if s, ok := p.(interface {
		Start(context.Context) error
	}); ok {
		if err := s.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}
	} else {
		t.Fatal("plugin does not implement Starter")
	}
}
