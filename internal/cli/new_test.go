package cli

import (
	"strings"
	"testing"
)

func TestParseDependencies(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  nil,
		},
		{
			name:  "single override",
			input: "react@19.2.5",
			want:  map[string]string{"react": "19.2.5"},
		},
		{
			name:  "multiple comma separated",
			input: "react@19.2.4,react-dom@19.2.4,tailwindcss@^4.3.0",
			want: map[string]string{
				"react":       "19.2.4",
				"react-dom":   "19.2.4",
				"tailwindcss": "^4.3.0",
			},
		},
		{
			name:  "tolerates whitespace around entries",
			input: " react@19.2.4 , tailwindcss@^4.3.0 ",
			want: map[string]string{
				"react":       "19.2.4",
				"tailwindcss": "^4.3.0",
			},
		},
		{
			name:  "scoped package",
			input: "@types/react@^19.3.0,@tailwindcss/cli@^4.3.0",
			want: map[string]string{
				"@types/react":     "^19.3.0",
				"@tailwindcss/cli": "^4.3.0",
			},
		},
		{
			name:  "npm alias version preserved",
			input: "react-server-dom-esm@npm:@kentcdodds/tmp-react-server-dom-esm@^19.0.1",
			want: map[string]string{
				"react-server-dom-esm": "npm:@kentcdodds/tmp-react-server-dom-esm@^19.0.1",
			},
		},
		{
			name:    "unknown package",
			input:   "raect@19.0.0",
			wantErr: `unknown dependency "raect"`,
		},
		{
			name:  "uiSpec package accepted",
			input: "@mui/material@^7.2.0,antd@^5.30.0",
			want: map[string]string{
				"@mui/material": "^7.2.0",
				"antd":          "^5.30.0",
			},
		},
		{
			name:    "missing version",
			input:   "react",
			wantErr: "invalid --dependencies entry",
		},
		{
			name:    "scoped package missing version",
			input:   "@types/react",
			wantErr: "invalid --dependencies entry",
		},
		{
			name:    "empty version after @",
			input:   "react@",
			wantErr: "invalid --dependencies entry",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDependencies(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (result: %v)", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
