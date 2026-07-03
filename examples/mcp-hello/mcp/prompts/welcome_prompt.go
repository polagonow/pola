package prompts

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
)

// WelcomePrompt is an MCP prompt. Add dependency fields here and resolve
// them in NewWelcomePrompt using core.MustInvoke[T](r).
type WelcomePrompt struct{}

// NewWelcomePrompt is discovered by the mcp autoload.
func NewWelcomePrompt(r *core.Registry) *WelcomePrompt {
	_ = r
	return &WelcomePrompt{}
}

func (p *WelcomePrompt) Prompt() *sdk.Prompt {
	return &sdk.Prompt{
		Name:        "welcome",
		Description: "TODO: describe this prompt",
	}
}

func (p *WelcomePrompt) Handle(ctx context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
	return &sdk.GetPromptResult{
		Messages: []*sdk.PromptMessage{
			{
				Role:    "user",
				Content: &sdk.TextContent{Text: "TODO: implement Welcome prompt"},
			},
		},
	}, nil
}
