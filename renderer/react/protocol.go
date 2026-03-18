//go:build react

package react

import (
	"fmt"
	"net/http"

	"github.com/polagonow/pola/core"
)

// RSCFlightProtocol implements helpers for the RSC Flight wire protocol.
type RSCFlightProtocol struct{}

// ContentType returns the MIME type for the RSC Flight wire format.
func (p *RSCFlightProtocol) ContentType() string { return ContentType }

// IsStreamingRequest reports whether the request carries the RSC Flight
// Content-Type header, meaning the client expects the raw stream.
func (p *RSCFlightProtocol) IsStreamingRequest(r *http.Request) bool {
	return IsStreamingRequest(r)
}

// Drain pulls all chunks from the render stream and writes them to w.
func (p *RSCFlightProtocol) Drain(vm core.SSRRuntime, handle core.StreamHandle, w core.StreamWriter) error {
	if handle == nil || handle.IsNil() {
		return fmt.Errorf("RSCFlightProtocol: stream handle is nil")
	}
	wroteAny, err := vm.DrainStream(handle, w)
	if err != nil {
		return err
	}
	if !wroteAny {
		return fmt.Errorf("RSCFlightProtocol: no Flight output written")
	}
	return nil
}
