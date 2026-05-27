package mcp

import (
	"context"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterToolAppends(t *testing.T) {
	resetForTests()

	RegisterTool(&sdk.Tool{Name: "a"}, func(_ context.Context, _ *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{}, nil
	})
	RegisterTool(&sdk.Tool{Name: "b"}, func(_ context.Context, _ *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{}, nil
	})

	tools, _, _, _, _ := snapshot()
	if len(tools) != 2 {
		t.Fatalf("snapshot: got %d tools, want 2", len(tools))
	}
	if tools[0].Tool.Name != "a" || tools[1].Tool.Name != "b" {
		t.Fatalf("order mismatch: %q, %q", tools[0].Tool.Name, tools[1].Tool.Name)
	}
}

func TestSnapshotDoesNotDrain(t *testing.T) {
	resetForTests()
	RegisterTool(&sdk.Tool{Name: "x"}, func(_ context.Context, _ *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{}, nil
	})

	first, _, _, _, _ := snapshot()
	second, _, _, _, _ := snapshot()
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("snapshot drained between calls: first=%d second=%d", len(first), len(second))
	}
}

func TestRegisterToolIsConcurrencySafe(t *testing.T) {
	resetForTests()

	const N = 100
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RegisterTool(&sdk.Tool{Name: "t"}, func(_ context.Context, _ *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				return &sdk.CallToolResult{}, nil
			})
		}()
	}
	wg.Wait()

	tools, _, _, _, _ := snapshot()
	if len(tools) != N {
		t.Fatalf("concurrent RegisterTool: got %d tools, want %d", len(tools), N)
	}
}

func TestRegisterResourceAndPrompt(t *testing.T) {
	resetForTests()

	RegisterResource(&sdk.Resource{URI: "test://r", Name: "r"}, func(_ context.Context, _ *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		return &sdk.ReadResourceResult{}, nil
	})
	RegisterPrompt(&sdk.Prompt{Name: "p"}, func(_ context.Context, _ *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		return &sdk.GetPromptResult{}, nil
	})

	_, res, _, prompts, _ := snapshot()
	if len(res) != 1 || res[0].Resource.URI != "test://r" {
		t.Fatalf("snapshot resources: %+v", res)
	}
	if len(prompts) != 1 || prompts[0].Prompt.Name != "p" {
		t.Fatalf("snapshot prompts: %+v", prompts)
	}
}
