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
			src, err := GenerateSource(opts, "", []string{"app/routes/health"}, nil, nil, nil, nil)
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
			// Validation is always on so c.Bind validates without per-handler code.
			if !strings.Contains(s, "validation.Plugin()") {
				t.Errorf("%s apiOnly=%v: missing validation.Plugin() registration", fw, apiOnly)
			}
			if !strings.Contains(s, `/validation"`) {
				t.Errorf("%s apiOnly=%v: missing validation import", fw, apiOnly)
			}
			// routes.Plugin() must no longer be generated.
			if strings.Contains(s, "routes.Plugin()") {
				t.Errorf("%s apiOnly=%v: legacy routes.Plugin() should not be generated", fw, apiOnly)
			}
		}
	}
}

// TestGenerateSource_DatabaseAdapters verifies the generated wiring passes the
// dialect/driver to the base plugin explicitly instead of blank-importing the
// adapter sub-package for init() side effects.
func TestGenerateSource_DatabaseAdapters(t *testing.T) {
	optionFor := map[string]string{
		"gorm":  "databasegorm.WithDialect(databaseadapter.Dialect())",
		"ent":   "databaseent.WithDriver(databaseadapter.Driver())",
		"beego": "databasebeego.WithDriver(databaseadapter.Driver())",
	}
	for orm, wantOption := range optionFor {
		for _, adapter := range []string{"sqlite", "postgresql", "mysql"} {
			opts := autoload.PluginOpts{
				PolaPackage:     "github.com/polagonow/pola",
				Database:        orm,
				DatabaseAdapter: adapter,
				APIOnly:         true,
			}
			src, err := GenerateSource(opts, "", nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatalf("%s/%s: generate: %v", orm, adapter, err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "pola_plugins.go", src, parser.AllErrors); err != nil {
				t.Fatalf("%s/%s: generated source does not parse: %v\n%s", orm, adapter, err, src)
			}
			s := string(src)
			adapterImport := `databaseadapter "github.com/polagonow/pola/database/` + orm + `/` + adapter + `"`
			if !strings.Contains(s, adapterImport) {
				t.Errorf("%s/%s: missing named adapter import %q\n%s", orm, adapter, adapterImport, s)
			}
			if !strings.Contains(s, wantOption) {
				t.Errorf("%s/%s: missing adapter option %q\n%s", orm, adapter, wantOption, s)
			}
			if strings.Contains(s, `_ "github.com/polagonow/pola/database/`) {
				t.Errorf("%s/%s: adapter must not be blank-imported\n%s", orm, adapter, s)
			}
		}
	}
}

// TestGenerateSource_DefaultFramework ensures an empty Framework defaults to std.
func TestGenerateSource_DefaultFramework(t *testing.T) {
	opts := autoload.PluginOpts{PolaPackage: "github.com/polagonow/pola", APIOnly: true}
	src, err := GenerateSource(opts, "", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "/webframework/std") {
		t.Errorf("empty Framework should default to std import")
	}
}
