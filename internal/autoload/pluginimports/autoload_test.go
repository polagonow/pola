package pluginimports

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/autoload"
)

// TestGenerateSource_Frameworks verifies the generated pola_plugins.go is valid
// Go and imports/registers the selected web framework, for each adapter and for
// both full-stack and API-only modes.
func TestGenerateSource_Frameworks(t *testing.T) {
	for _, fw := range []string{"std", "gin", "echo", "chi"} {
		for _, apiOnly := range []bool{false, true} {
			opts := autoload.PluginOpts{
				PolaPackage: "github.com/polagonow/pola",
				Framework:   fw,
				Engine:      "goja",
				Bundler:     "esbuild",
				Renderer:    "react",
				Router:      "nextjs",
				APIOnly:     apiOnly,
			}
			src, err := GenerateSource(opts, "", []string{"app/routes/health"}, nil, nil, nil)
			if err != nil {
				t.Fatalf("%s apiOnly=%v: generate: %v", fw, apiOnly, err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "pola_plugins.go", src, parser.AllErrors); err != nil {
				t.Fatalf("%s apiOnly=%v: generated source does not parse: %v\n%s", fw, apiOnly, err, src)
			}
			s := string(src)
			if !strings.Contains(s, "/webframework/"+fw) {
				t.Errorf("%s apiOnly=%v: missing webframework import\n%s", fw, apiOnly, s)
			}
			if !strings.Contains(s, "webfw.Plugin()") {
				t.Errorf("%s apiOnly=%v: missing webfw.Plugin() registration", fw, apiOnly)
			}
			// routes.Plugin() must no longer be generated.
			if strings.Contains(s, "routes.Plugin()") {
				t.Errorf("%s apiOnly=%v: legacy routes.Plugin() should not be generated", fw, apiOnly)
			}
		}
	}
}

// TestGenerateSource_DefaultFramework ensures an empty Framework defaults to std.
func TestGenerateSource_DefaultFramework(t *testing.T) {
	opts := autoload.PluginOpts{PolaPackage: "github.com/polagonow/pola", APIOnly: true}
	src, err := GenerateSource(opts, "", nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "/webframework/std") {
		t.Errorf("empty Framework should default to std import")
	}
}
