package routes

import "sync"

// funcReg is a queued function-based route registration.
type funcReg struct {
	method string
	fn     any
}

var (
	funcMu      sync.Mutex
	funcPending []funcReg
)

// RegisterFunc queues a function-based route handler. method is the HTTP verb
// (GET, POST, …) and fn must have the handler signature:
//
//	func(core.Context) error
//
// The URL pattern is derived from fn's package path, identically to struct
// routes. Must be called from an init() function (typically from the generated
// pola_route_init.go overlay).
func RegisterFunc(method string, fn any) {
	funcMu.Lock()
	defer funcMu.Unlock()
	funcPending = append(funcPending, funcReg{method: method, fn: fn})
}

// drainFuncs returns and clears the pending function registrations.
func drainFuncs() []funcReg {
	funcMu.Lock()
	defer funcMu.Unlock()
	out := funcPending
	funcPending = nil
	return out
}
