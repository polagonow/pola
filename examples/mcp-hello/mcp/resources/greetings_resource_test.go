package resources

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
)

func TestGreetingsResource_DescribesItself(t *testing.T) {
	r := core.NewRegistry()
	res := NewGreetingsResource(r)
	descriptor := res.Resource()
	if descriptor == nil {
		t.Fatal("Resource() returned nil")
	}
	if descriptor.URI != "pola://greetings" {
		t.Fatalf("URI: got %q want %q", descriptor.URI, "pola://greetings")
	}
}

func TestGreetingsResource_HandleReturnsResult(t *testing.T) {
	r := core.NewRegistry()
	res := NewGreetingsResource(r)
	req := &sdk.ReadResourceRequest{Params: &sdk.ReadResourceParams{URI: "pola://greetings"}}
	result, err := res.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
