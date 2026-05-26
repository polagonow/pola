package react

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

)

// ClientRef is the wire representation of a Client Component reference.
// The browser's RSC runtime resolves this to the actual React component
// by looking up the manifest produced at build time.
type ClientRef struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Chunks []string `json:"chunks"`
	Async  bool     `json:"async"`
}

// ClientManifest maps moduleId → ClientRef.
type ClientManifest map[string]ClientRef

// ChunkType represents the type prefix in the RSC Flight wire format.
// Each line on the wire looks like:   <id>:<type><data>\n
type ChunkType string

// Flight chunk type constants for the RSC wire protocol.
const (
	ChunkJSON     ChunkType = "J" // serialised React element tree
	ChunkModule   ChunkType = "I" // client component reference (module)
	ChunkHint     ChunkType = "H" // preload / prefetch hint
	ChunkError    ChunkType = "E" // error boundary payload
	ChunkSuspense ChunkType = "S" // suspense placeholder
)

// FlightWriter encodes and streams RSC Flight Protocol chunks.
// It is safe for concurrent use – multiple goroutines (one per Suspense
// boundary) call WriteChunk simultaneously and the writer serialises them.
type FlightWriter struct {
	w       io.Writer
	flusher http.Flusher
	mu      sync.Mutex
	counter atomic.Int32
}

// NextID returns a monotonically increasing chunk ID.
func (fw *FlightWriter) NextID() int {
	return int(fw.counter.Add(1)) - 1
}

// WriteChunk serialises data as JSON and writes one Flight line.
// Thread-safe – safe to call from multiple goroutines.
func (fw *FlightWriter) WriteChunk(id int, t ChunkType, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("flight: marshal chunk %d: %w", id, err)
	}
	fw.mu.Lock()
	defer fw.mu.Unlock()
	_, err = fmt.Fprintf(fw.w, "%d:%s%s\n", id, t, b)
	if err != nil {
		return err
	}
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return nil
}

// WriteError streams an error chunk for the given suspense slot.
func (fw *FlightWriter) WriteError(id int, msg string) error {
	return fw.WriteChunk(id, ChunkError, map[string]string{"message": msg})
}

// WriteModuleRef streams a client component reference chunk.
// The React client uses this to resolve which JS bundle to load.
func (fw *FlightWriter) WriteModuleRef(id int, ref ClientRef) error {
	return fw.WriteChunk(id, ChunkModule, ref)
}

// WriteSuspensePlaceholder writes an immediate placeholder so the shell
// renders instantly. The real content is streamed later via WriteChunk.
func (fw *FlightWriter) WriteSuspensePlaceholder(id int, fallback any) error {
	return fw.WriteChunk(id, ChunkSuspense, map[string]any{
		"id":       id,
		"fallback": fallback,
	})
}

// WriteRaw writes raw bytes directly to the underlying writer without any
// encoding. Used when react-server-dom-webpack has already produced the
// Flight wire format — we just pipe the bytes through.
func (fw *FlightWriter) WriteRaw(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.w.Write(p)
}

// Flush pushes any buffered bytes to the client.
func (fw *FlightWriter) Flush() {
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
}
