package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// ChunkType represents the type prefix in the RSC Flight wire format.
// Each line on the wire looks like:   <id>:<type><data>\n
type ChunkType string

const (
	ChunkJSON      ChunkType = "J" // serialised React element tree
	ChunkModule    ChunkType = "I" // client component reference (module)
	ChunkHint      ChunkType = "H" // preload / prefetch hint
	ChunkError     ChunkType = "E" // error boundary payload
	ChunkSuspense  ChunkType = "S" // suspense placeholder
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

// NewFlightWriter wraps an http.ResponseWriter that must also implement
// http.Flusher (it always does in net/http).
func NewFlightWriter(w http.ResponseWriter) *FlightWriter {
	fw := &FlightWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		fw.flusher = f
	}
	return fw
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
		return fmt.Errorf("flight: marshal chunk %d: %w", err, err)
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

// ClientRef is the wire representation of a Client Component reference.
// The browser's RSC runtime resolves this to the actual React component
// by looking up the manifest that esbuild produced at build time.
type ClientRef struct {
	ID     string   `json:"id"`     // stable export identifier
	Name   string   `json:"name"`   // named export (or "default")
	Chunks []string `json:"chunks"` // JS bundle filenames to preload
	Async  bool     `json:"async"`  // whether the chunk is lazy-loaded
}

// ReactNode is the Go-side representation of a React virtual DOM node.
// Server Components render these; they are then JSON-encoded into the
// Flight payload that React's client runtime reconciles against the DOM.
type ReactNode struct {
	Type     any            `json:"type"`               // string tag or ClientRef
	Key      *string        `json:"key,omitempty"`
	Ref      *string        `json:"ref,omitempty"`
	Props    map[string]any `json:"props"`
}

// TextNode is a special sentinel for raw text content.
type TextNode struct {
	Text string
}

// NullNode represents React null (renders nothing).
type NullNode struct{}

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

// NewFlightWriterBuffer creates a FlightWriter backed by a strings.Builder
// for collecting server-rendered HTML into memory (used by handlePage).
func NewFlightWriterBuffer(buf interface{ WriteString(string) (int, error); Write([]byte) (int, error) }) *FlightWriter {
	return &FlightWriter{w: buf}
}
