package react

import (
	"fmt"
	"sync"

	"github.com/polagonow/pola/framework/globals"

	gojalib "github.com/dop251/goja"
)

// SuspenseResult is the outcome of resolving one async Suspense boundary.
type SuspenseResult struct {
	ID    int    // chunk ID that was reserved for this boundary
	Node  any    // resolved React node (to be JSON-encoded into the stream)
	Error string // non-empty if the async render failed
}

// SuspenseBoundary represents a single <Suspense> node encountered during
// the synchronous tree walk. Its content fn is async (returns a Promise in JS)
// so we schedule it as a goroutine and stream the result when it resolves.
type SuspenseBoundary struct {
	ID       int                 // pre-assigned flight chunk ID
	Fallback any                 // what to show while loading
	Fn       func() (any, error) // the async render thunk
}

// SuspenseScheduler collects all Suspense boundaries encountered during the
// initial synchronous render pass, then fans them out as goroutines and
// streams each result as it resolves.
type SuspenseScheduler struct {
	fw         *FlightWriter
	boundaries []SuspenseBoundary
	mu         sync.Mutex
}

// NewSuspenseScheduler creates a scheduler attached to the given writer.
func NewSuspenseScheduler(fw *FlightWriter) *SuspenseScheduler {
	return &SuspenseScheduler{fw: fw}
}

// Register adds a Suspense boundary. Called during the synchronous render walk
// when a node tagged as async is encountered. Returns the reserved chunk ID.
func (s *SuspenseScheduler) Register(fallback any, fn func() (any, error)) int {
	id := s.fw.NextID()
	s.mu.Lock()
	s.boundaries = append(s.boundaries, SuspenseBoundary{
		ID:       id,
		Fallback: fallback,
		Fn:       fn,
	})
	s.mu.Unlock()
	return id
}

// FlushAll fans out all registered boundaries as goroutines and streams each
// result back via the Flight writer as it resolves. Blocks until all are done.
func (s *SuspenseScheduler) FlushAll() {
	if len(s.boundaries) == 0 {
		return
	}

	results := make(chan SuspenseResult, len(s.boundaries))
	var wg sync.WaitGroup

	for _, b := range s.boundaries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			node, err := b.Fn()
			if err != nil {
				results <- SuspenseResult{ID: b.ID, Error: err.Error()}
				return
			}
			results <- SuspenseResult{ID: b.ID, Node: node}
		}()
	}

	// Close channel once all goroutines finish so the range below exits.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Stream each result as it arrives – first resolved = first flushed.
	for res := range results {
		if res.Error != "" {
			_ = s.fw.WriteError(res.ID, res.Error)
		} else {
			_ = s.fw.WriteChunk(res.ID, ChunkJSON, res.Node)
		}
	}
}

// PromiseToGo resolves a Goja Promise synchronously by running the JS event
// loop until the promise settles. This is the bridge between JS async/await
// (which Goja doesn't execute automatically) and Go goroutines.
//
// Each Suspense boundary goroutine calls this with the Promise returned by
// the async Server Component function.
func PromiseToGo(rt *gojalib.Runtime, promise *gojalib.Promise) (gojalib.Value, error) {
	// Goja's event loop must be driven manually. We spin until settled.
	// For production you'd use goja_nodejs's event loop; this minimal version
	// handles the straightforward case of a single pending microtask chain.
	for promise.State() == gojalib.PromiseStatePending {
		// Run any queued microtasks (then-callbacks, async continuations).
		if err := rt.GlobalObject().Get(globals.RunMicrotasksFn); err != nil {
			break
		}
	}

	switch promise.State() {
	case gojalib.PromiseStateFulfilled:
		return promise.Result(), nil
	case gojalib.PromiseStateRejected:
		return nil, fmt.Errorf("promise rejected: %v", promise.Result().Export())
	default:
		return nil, fmt.Errorf("promise still pending after event loop drain")
	}
}
