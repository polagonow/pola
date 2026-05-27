package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolProvider is implemented by user-defined tools that should be auto-
// registered with the MCP server. The autoload scanner under
// internal/autoload/mcp discovers New*Tool constructors in mcp/tools/
// and wires their resulting *T into the DI container; the generated plugin
// then resolves each one and registers it via Server.AddTool.
type ToolProvider interface {
	Tool() *sdk.Tool
	Handle(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error)
}

// ResourceProvider mirrors ToolProvider for MCP resources.
type ResourceProvider interface {
	Resource() *sdk.Resource
	Handle(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error)
}

// PromptProvider mirrors ToolProvider for MCP prompts.
type PromptProvider interface {
	Prompt() *sdk.Prompt
	Handle(ctx context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error)
}

// ToolEntry is what RegisterTool stores in the package-level registry.
type ToolEntry struct {
	Tool    *sdk.Tool
	Handler sdk.ToolHandler
}

// ResourceEntry is what RegisterResource stores in the package-level registry.
type ResourceEntry struct {
	Resource *sdk.Resource
	Handler  sdk.ResourceHandler
}

// ResourceTemplateEntry is what RegisterResourceTemplate stores.
type ResourceTemplateEntry struct {
	Template *sdk.ResourceTemplate
	Handler  sdk.ResourceHandler
}

// PromptEntry is what RegisterPrompt stores in the package-level registry.
type PromptEntry struct {
	Prompt  *sdk.Prompt
	Handler sdk.PromptHandler
}
