package mcp

import (
	"context"
	"slices"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	regMu      sync.Mutex
	pkgTools   []ToolEntry
	pkgRes     []ResourceEntry
	pkgResTpl  []ResourceTemplateEntry
	pkgPrompts []PromptEntry

	pkgTypedTools []func(s *sdk.Server)
)

// RegisterTool adds a tool to the package-level registry. The MCP plugin's
// Start hook applies it to the server. Safe to call from init().
func RegisterTool(t *sdk.Tool, h sdk.ToolHandler) {
	regMu.Lock()
	pkgTools = append(pkgTools, ToolEntry{Tool: t, Handler: h})
	regMu.Unlock()
}

// RegisterTypedTool registers a tool with typed input/output. The SDK derives
// JSON Schema from the In and Out types. Safe to call from init().
func RegisterTypedTool[In, Out any](
	t *sdk.Tool,
	h func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error),
) {
	regMu.Lock()
	pkgTypedTools = append(pkgTypedTools, func(s *sdk.Server) {
		sdk.AddTool(s, t, h)
	})
	regMu.Unlock()
}

// RegisterResource adds a resource to the package-level registry.
func RegisterResource(r *sdk.Resource, h sdk.ResourceHandler) {
	regMu.Lock()
	pkgRes = append(pkgRes, ResourceEntry{Resource: r, Handler: h})
	regMu.Unlock()
}

// RegisterResourceTemplate adds a templated resource to the package-level registry.
func RegisterResourceTemplate(t *sdk.ResourceTemplate, h sdk.ResourceHandler) {
	regMu.Lock()
	pkgResTpl = append(pkgResTpl, ResourceTemplateEntry{Template: t, Handler: h})
	regMu.Unlock()
}

// RegisterPrompt adds a prompt to the package-level registry.
func RegisterPrompt(p *sdk.Prompt, h sdk.PromptHandler) {
	regMu.Lock()
	pkgPrompts = append(pkgPrompts, PromptEntry{Prompt: p, Handler: h})
	regMu.Unlock()
}

// snapshot returns a copy of every package-level registration. It does NOT
// clear the slices, so dev-mode hot rebuilds that re-run plugin.Start (without
// re-running init()) still see the same entries.
func snapshot() (
	tools []ToolEntry,
	res []ResourceEntry,
	resTpl []ResourceTemplateEntry,
	prompts []PromptEntry,
	typed []func(*sdk.Server),
) {
	regMu.Lock()
	defer regMu.Unlock()
	return slices.Clone(pkgTools),
		slices.Clone(pkgRes),
		slices.Clone(pkgResTpl),
		slices.Clone(pkgPrompts),
		slices.Clone(pkgTypedTools)
}

// resetForTests clears the package-level registry. Tests use this to start
// from a clean slate.
func resetForTests() {
	regMu.Lock()
	defer regMu.Unlock()
	pkgTools = nil
	pkgRes = nil
	pkgResTpl = nil
	pkgPrompts = nil
	pkgTypedTools = nil
}
