package react

import (
	"fmt"
	"net/http"

	"gojsx/framework"
	vmgoja "gojsx/vm/goja"
)

// RSCFlightProtocol implements framework.StreamProtocol for React Server
// Components using the RSC Flight wire format. It is paired with GojaVM —
// Drain type-asserts both vm and handle to their Goja-specific types.
type RSCFlightProtocol struct{}

// ContentType returns the MIME type for the RSC Flight wire format.
func (p *RSCFlightProtocol) ContentType() string { return "text/x-component" }

// IsStreamingRequest reports whether the request carries the RSC Flight
// Content-Type header, meaning the client expects the raw stream.
func (p *RSCFlightProtocol) IsStreamingRequest(r *http.Request) bool {
	return r.Header.Get("Content-Type") == "text/x-component"
}

// Drain pulls all chunks from the Goja render stream and writes them to w.
// Both vm and handle must be the Goja-specific concrete types produced by
// GojaVMFactory / VM.CallRenderFunction.
func (p *RSCFlightProtocol) Drain(vm framework.VM, handle framework.StreamHandle, w framework.StreamWriter) error {
	gojaVM, ok := vm.(*vmgoja.VM)
	if !ok {
		return fmt.Errorf("RSCFlightProtocol: expected *vmgoja.VM, got %T", vm)
	}
	gojaHandle, ok := handle.(*vmgoja.GojaStreamHandle)
	if !ok {
		return fmt.Errorf("RSCFlightProtocol: expected *vmgoja.GojaStreamHandle, got %T", handle)
	}
	if gojaHandle.IsNil() {
		return fmt.Errorf("RSCFlightProtocol: stream handle is nil")
	}

	wroteAny, err := vmgoja.DrainStream(gojaVM, w, gojaHandle.Sess)
	if err != nil {
		return err
	}
	if !wroteAny {
		return fmt.Errorf("RSCFlightProtocol: no Flight output written")
	}
	return nil
}
