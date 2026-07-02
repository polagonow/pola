package resources

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/polagonow/pola/core"
)

// GreetingsResource is an MCP resource. Add dependency fields here and resolve
// them in NewGreetingsResource using core.MustInvoke[T](r).
type GreetingsResource struct{}

// NewGreetingsResource is discovered by the mcp autoload.
func NewGreetingsResource(r *core.Registry) *GreetingsResource {
	_ = r
	return &GreetingsResource{}
}

func (r *GreetingsResource) Resource() *sdk.Resource {
	return &sdk.Resource{
		URI:         "pola://greetings",
		Name:        "Greetings",
		Description: "TODO: describe this resource",
		MIMEType:    "application/json",
	}
}

func (r *GreetingsResource) Handle(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
	return &sdk.ReadResourceResult{
		Contents: []*sdk.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: `{"message":"TODO: implement Greetings"}`},
		},
	}, nil
}
