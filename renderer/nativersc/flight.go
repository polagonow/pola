// Package nativersc is a Go-driven React Server Components renderer for Pola.
//
// Unlike the react renderer — which delegates the entire Flight serialization to
// react-server-dom-webpack's renderToReadableStream running inside goja and pipes
// the resulting bytes straight through — nativersc executes server components in
// goja but walks the resulting React element tree and serializes the RSC Flight
// wire format natively in Go. This gives full control over the protocol,
// streaming, and Suspense.
//
// The wire format targets react-server-dom-esm/client, the parser the browser
// actually runs (renderer/react keeps the same @pola/react/client entry). Each
// row is one line:
//
//	<hexId>:<tag?><payload>\n
//
// Model rows are UNTAGGED (<hex>:<json>); "I" rows are client-component imports;
// "E" rows are errors. References inside a model are $-prefixed strings:
// "$" (element-tuple marker), "$L<hex>" (lazy/client chunk), "$<hex>" (outlined
// model), "$@<hex>" (promise), "$S<name>" (symbol, e.g. $Sreact.suspense),
// "$undefined", "$D<iso>" (Date), "$n<digits>" (BigInt). A user string that
// itself starts with "$" is escaped to "$$…".
package nativersc

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

// elementMarker is the head of a serialized React element tuple
// ["$", type, key, props]. It marshals to the JSON string "$", which the
// react-server-dom-esm client decodes to REACT_ELEMENT_TYPE.
type elementMarker struct{}

func (elementMarker) MarshalJSON() ([]byte, error) { return []byte(`"$"`), nil }

// ref is an in-model reference to another Flight row. A lazy ref ("$L<hex>") sits
// in the element-type position for client components and outlined elements; a
// plain ref ("$<hex>") points at an outlined model value.
type ref struct {
	lazy bool
	id   int
}

func (r ref) MarshalJSON() ([]byte, error) {
	if r.lazy {
		return []byte(`"$L` + strconv.FormatInt(int64(r.id), 16) + `"`), nil
	}
	return []byte(`"$` + strconv.FormatInt(int64(r.id), 16) + `"`), nil
}

// symbolRef is an in-model symbol reference, e.g. "$Sreact.suspense".
type symbolRef struct{ name string }

func (s symbolRef) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal("$S" + s.name)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// promiseRef is an in-model promise reference ("$@<hex>") whose resolved value
// streams later as model row <hex>.
type promiseRef struct{ id int }

func (p promiseRef) MarshalJSON() ([]byte, error) {
	return []byte(`"$@` + strconv.FormatInt(int64(p.id), 16) + `"`), nil
}

// serverRef is an in-model server-reference ("$F<hex>") pointing at a metadata
// row {id, bound}. The react-server-dom-esm client decodes it into a callable
// that invokes response._callServer(metadata.id, args).
type serverRef struct{ id int }

func (s serverRef) MarshalJSON() ([]byte, error) {
	return []byte(`"$F` + strconv.FormatInt(int64(s.id), 16) + `"`), nil
}

// flightWriter encodes and streams RSC Flight rows to an underlying writer.
//
// It is safe for concurrent use: from Phase 2 on, Suspense boundaries stream
// from multiple goroutines, so row writes are serialized under mu and row ids
// are handed out atomically.
type flightWriter struct {
	w       io.Writer
	flusher interface{ Flush() }
	mu      sync.Mutex
	counter atomic.Int64
}

func newFlightWriter(w io.Writer) *flightWriter {
	fw := &flightWriter{w: w}
	if f, ok := w.(interface{ Flush() }); ok {
		fw.flusher = f
	}
	return fw
}

// nextID returns a monotonically increasing, 0-based row id.
func (fw *flightWriter) nextID() int { return int(fw.counter.Add(1) - 1) }

// writeRow writes one Flight row: "<hex>:<tag><payload>\n".
func (fw *flightWriter) writeRow(id int, tag string, payload []byte) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	b := make([]byte, 0, len(tag)+len(payload)+8)
	b = strconv.AppendInt(b, int64(id), 16)
	b = append(b, ':')
	b = append(b, tag...)
	b = append(b, payload...)
	b = append(b, '\n')
	if _, err := fw.w.Write(b); err != nil {
		return err
	}
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return nil
}

// writeModel marshals model and writes it as an untagged model row.
func (fw *flightWriter) writeModel(id int, model any) error {
	b, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("nativersc: marshal model row %d: %w", id, err)
	}
	return fw.writeRow(id, "", b)
}

// writeImport writes a client-component import row: <hex>:I["<url>","<export>"].
// The react-server-dom-esm client reads metadata[0] as the importable URL and
// metadata[1] as the export name.
func (fw *flightWriter) writeImport(id int, url, export string) error {
	b, err := json.Marshal([2]string{url, export})
	if err != nil {
		return fmt.Errorf("nativersc: marshal import row %d: %w", id, err)
	}
	return fw.writeRow(id, "I", b)
}

// writeServerReference writes the metadata row for a server reference (action),
// referenced from a model as "$F<hex>". The react-server-dom-esm client reads
// metadata.id and calls response._callServer(id, args); bound is null because
// nativersc does not pre-bind arguments.
func (fw *flightWriter) writeServerReference(id int, actionID string) error {
	b, err := json.Marshal(map[string]any{"id": actionID, "bound": nil})
	if err != nil {
		return fmt.Errorf("nativersc: marshal server reference row %d: %w", id, err)
	}
	return fw.writeRow(id, "", b)
}

// writeError writes an error row: <hex>:E{"digest":…,"message":…,"stack":[…]}.
func (fw *flightWriter) writeError(id int, digest, message string, stack []string) error {
	if stack == nil {
		stack = []string{}
	}
	b, err := json.Marshal(map[string]any{
		"digest":  digest,
		"message": message,
		"stack":   stack,
	})
	if err != nil {
		return fmt.Errorf("nativersc: marshal error row %d: %w", id, err)
	}
	return fw.writeRow(id, "E", b)
}

func (fw *flightWriter) flush() {
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
}
