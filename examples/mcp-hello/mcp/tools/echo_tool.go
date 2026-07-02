package tools

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	polamcp "github.com/polagonow/pola/mcp"
)

// EchoIn is the typed input for the tool. Field tags become JSON Schema.
type EchoIn struct {
	// Example field; rename or remove as needed.
	Message string `json:"message" jsonschema:"the message to echo"`
}

// EchoOut is the typed output for the tool.
type EchoOut struct {
	Echo string `json:"echo"`
}

func init() {
	polamcp.RegisterTypedTool[
		EchoIn,
		EchoOut,
	](
		&sdk.Tool{
			Name:        "echo",
			Description: "TODO: describe what Echo does",
		},
		func(_ context.Context, _ *sdk.CallToolRequest, in EchoIn) (*sdk.CallToolResult, EchoOut, error) {
			return nil, EchoOut{Echo: in.Message}, nil
		},
	)
}
