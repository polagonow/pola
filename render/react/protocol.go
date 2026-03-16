package react

import (
	"fmt"
	"net/http"

	"gojsx/framework"
)

// streamDrainable is satisfied by any VM that can drain its own stream handle.
// Both vmgoja.VM and vmv8go.V8VM implement this interface.
type streamDrainable interface {
	DrainStream(handle framework.StreamHandle, w framework.StreamWriter) (bool, error)
}

// RSCFlightProtocol implements framework.StreamProtocol for React Server
// Components using the RSC Flight wire format.
type RSCFlightProtocol struct{}

// ContentType returns the MIME type for the RSC Flight wire format.
func (p *RSCFlightProtocol) ContentType() string { return "text/x-component" }

// IsStreamingRequest reports whether the request carries the RSC Flight
// Content-Type header, meaning the client expects the raw stream.
func (p *RSCFlightProtocol) IsStreamingRequest(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "text/x-component"
}

// Drain pulls all chunks from the render stream and writes them to w.
// The vm must implement the streamDrainable interface (any VM produced by
// a VMFactory that registers DrainStream satisfies this).
func (p *RSCFlightProtocol) Drain(vm framework.VM, handle framework.StreamHandle, w framework.StreamWriter) error {
	drainable, ok := vm.(streamDrainable)
	if !ok {
		return fmt.Errorf("RSCFlightProtocol: VM %T does not implement DrainStream", vm)
	}
	if handle.IsNil() {
		return fmt.Errorf("RSCFlightProtocol: stream handle is nil")
	}
	wroteAny, err := drainable.DrainStream(handle, w)
	if err != nil {
		return err
	}
	if !wroteAny {
		return fmt.Errorf("RSCFlightProtocol: no Flight output written")
	}
	return nil
}
