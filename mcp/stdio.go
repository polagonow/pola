package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// runStdio launches server.Run with a StdioTransport in a goroutine.
// It returns a cancel func that the plugin's Shutdown hook calls to
// terminate the transport. server.Run's error (if any) is reported via errCh.
func runStdio(srv *sdk.Server) (cancel context.CancelFunc, errCh <-chan error) {
	ctx, cancelFn := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() {
		ch <- srv.Run(ctx, &sdk.StdioTransport{})
		close(ch)
	}()
	return cancelFn, ch
}
