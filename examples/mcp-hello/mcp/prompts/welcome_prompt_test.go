package prompts

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
)

func TestWelcomePrompt_DescribesItself(t *testing.T) {
	r := core.NewRegistry()
	p := NewWelcomePrompt(r)
	descriptor := p.Prompt()
	if descriptor == nil {
		t.Fatal("Prompt() returned nil")
	}
	if descriptor.Name != "welcome" {
		t.Fatalf("Name: got %q want %q", descriptor.Name, "welcome")
	}
}

func TestWelcomePrompt_HandleReturnsResult(t *testing.T) {
	r := core.NewRegistry()
	p := NewWelcomePrompt(r)
	result, err := p.Handle(context.Background(), &sdk.GetPromptRequest{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Messages) == 0 {
		t.Fatal("expected at least one message")
	}
}
