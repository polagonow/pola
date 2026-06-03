package nativersc

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestMarshalers(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want string
	}{
		{"element marker", elementMarker{}, `"$"`},
		{"lazy ref", ref{lazy: true, id: 1}, `"$L1"`},
		{"lazy ref hex", ref{lazy: true, id: 26}, `"$L1a"`},
		{"outline ref", ref{lazy: false, id: 3}, `"$3"`},
		{"symbol ref", symbolRef{name: "react.suspense"}, `"$Sreact.suspense"`},
		{"promise ref", promiseRef{id: 4}, `"$@4"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.val)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Fatalf("got %s, want %s", b, tc.want)
			}
		})
	}
}

// TestClientElementModel verifies a host element containing a client component
// reference serializes to the exact wire shape the esm client expects.
func TestClientElementModel(t *testing.T) {
	clientEl := []any{elementMarker{}, ref{lazy: true, id: 1}, nil, map[string]any{"initial": 3}}
	mainEl := []any{elementMarker{}, "main", nil, map[string]any{"children": clientEl}}

	b, err := json.Marshal(mainEl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `["$","main",null,{"children":["$","$L1",null,{"initial":3}]}]`
	if string(b) != want {
		t.Fatalf("got  %s\nwant %s", b, want)
	}
}

// TestFlightWriterOutput verifies the full row stream: an import row (hex id,
// I-tag, 2-tuple) followed by an untagged model row referencing it.
func TestFlightWriterOutput(t *testing.T) {
	var buf bytes.Buffer
	fw := newFlightWriter(&buf)

	modelID := fw.nextID() // reserve row 0 for the root model
	refID := fw.nextID()   // row 1 for the client import

	if err := fw.writeImport(refID, "/public/assets/Counter-a1b2.js", "Counter"); err != nil {
		t.Fatal(err)
	}
	clientEl := []any{elementMarker{}, ref{lazy: true, id: refID}, nil, map[string]any{"initial": 3}}
	mainEl := []any{elementMarker{}, "main", nil, map[string]any{"children": clientEl}}
	if err := fw.writeModel(modelID, mainEl); err != nil {
		t.Fatal(err)
	}

	want := "1:I[\"/public/assets/Counter-a1b2.js\",\"Counter\"]\n" +
		"0:[\"$\",\"main\",null,{\"children\":[\"$\",\"$L1\",null,{\"initial\":3}]}]\n"
	if buf.String() != want {
		t.Fatalf("got:\n%q\nwant:\n%q", buf.String(), want)
	}
}

// TestWriteError verifies the E-row shape.
func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	fw := newFlightWriter(&buf)
	if err := fw.writeError(4, "err_7f3", "boom", nil); err != nil {
		t.Fatal(err)
	}
	want := "4:E{\"digest\":\"err_7f3\",\"message\":\"boom\",\"stack\":[]}\n"
	if buf.String() != want {
		t.Fatalf("got:\n%q\nwant:\n%q", buf.String(), want)
	}
}
