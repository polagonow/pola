package tools

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"

	"mcp-hello/repositories"
	"mcp-hello/services"
)

// GreetingTool exposes the scaffolded GreetingService as an MCP tool so AI
// clients (e.g. Claude Desktop) can list and create greetings stored in the
// app's database. It demonstrates DI: the constructor pulls the service out
// of the framework's container, no manual wiring needed.
type GreetingTool struct {
	svc *services.GreetingService
}

func NewGreetingTool(r *core.Registry) *GreetingTool {
	return &GreetingTool{
		svc: core.MustInvoke[*services.GreetingService](r),
	}
}

func (t *GreetingTool) Tool() *sdk.Tool {
	return &sdk.Tool{
		Name:        "greeting",
		Description: "List greetings or create a new one. Pass {\"create\":\"hello\"} to create; otherwise lists.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"create": map[string]any{
					"type":        "string",
					"description": "Optional message to create. If omitted, the tool lists existing greetings.",
				},
			},
		},
	}
}

func (t *GreetingTool) Handle(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
	var args struct {
		Create string `json:"create"`
	}
	if raw := req.Params.Arguments; raw != nil {
		_ = json.Unmarshal(raw, &args)
	}

	if args.Create != "" {
		g := &repositories.Greeting{Message: args.Create}
		if err := t.svc.Create(ctx, g); err != nil {
			return nil, fmt.Errorf("create greeting: %w", err)
		}
		return &sdk.CallToolResult{
			Content: []sdk.Content{
				&sdk.TextContent{Text: fmt.Sprintf("created greeting #%d: %q", g.ID, g.Message)},
			},
		}, nil
	}

	page, err := t.svc.List(ctx, repository.ListParams{PerPage: 50})
	if err != nil {
		return nil, fmt.Errorf("list greetings: %w", err)
	}
	text := fmt.Sprintf("%d greeting(s):\n", len(page.Items))
	for _, g := range page.Items {
		text += fmt.Sprintf("  #%d: %s\n", g.ID, g.Message)
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: text}},
	}, nil
}
