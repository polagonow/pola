package entclient

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/autoload"
)

const polaPkg = "github.com/polagonow/pola"

// contribute runs the autoloader against a temp project. Returns the generated
// pola_ent_client_plugin.go source, if any was emitted.
func contribute(t *testing.T, disco *autoload.RepoDiscovery) (src string, discovered *autoload.EntClientDiscovery) {
	t.Helper()
	projectDir := t.TempDir()
	tmpDir := t.TempDir()
	ctx := &autoload.Context{
		ProjectDir: projectDir,
		TmpDir:     tmpDir,
		Opts:       autoload.PluginOpts{PolaPackage: polaPkg},
		Replace:    map[string]string{},
		Discovery:  &autoload.Discovery{RepoDisco: disco},
	}
	if err := (&autoloadImpl{}).Contribute(ctx); err != nil {
		t.Fatalf("contribute: %v", err)
	}
	discovered = ctx.Discovery.EntClientDisco
	for _, real := range ctx.Replace {
		b, err := os.ReadFile(real)
		if err != nil {
			t.Fatalf("read generated: %v", err)
		}
		src = string(b)
		if _, err := parser.ParseFile(token.NewFileSet(), filepath.Base(real), src, parser.AllErrors); err != nil {
			t.Fatalf("generated does not parse: %v\n%s", err, src)
		}
	}
	return src, discovered
}

func TestEntClient_EmitsForEnt(t *testing.T) {
	src, disco := contribute(t, &autoload.RepoDiscovery{
		ORM:          "ent",
		ModulePath:   "myapp",
		EntClientDir: "db/client/ent",
		RepoDir:      "repositories",
		PkgName:      "ent",
		ImportPath:   "myapp/repositories/ent",
		Repositories: []autoload.PluginEntry{{Name: "Todo"}},
	})
	if disco == nil {
		t.Fatal("want EntClientDisco populated, got nil")
	}
	if disco.PkgName != "ent" {
		t.Errorf("PkgName: got %q, want %q", disco.PkgName, "ent")
	}
	checks := []string{
		`func EntClientPlugin() core.Plugin`,
		`"myapp/db/client/ent"`,
		`entsql "entgo.io/ent/dialect/sql"`,
		`ent.NewClient(ent.Driver(drv))`,
		`core.Provide[*ent.Client](r`,
	}
	for _, want := range checks {
		if !strings.Contains(src, want) {
			t.Errorf("want %q in output\n%s", want, src)
		}
	}
}

func TestEntClient_SkipsForGorm(t *testing.T) {
	src, disco := contribute(t, &autoload.RepoDiscovery{
		ORM: "gorm", ModulePath: "myapp", RepoDir: "repositories", PkgName: "gorm",
		Repositories: []autoload.PluginEntry{{Name: "Todo"}},
	})
	if src != "" {
		t.Errorf("gorm project should not emit ent client plugin\n%s", src)
	}
	if disco != nil {
		t.Errorf("gorm project should not set EntClientDisco, got %+v", disco)
	}
}

func TestEntClient_SkipsWhenNoRepos(t *testing.T) {
	_, disco := contribute(t, &autoload.RepoDiscovery{
		ORM: "ent", ModulePath: "myapp", EntClientDir: "db/client/ent",
		RepoDir: "repositories", PkgName: "ent",
	})
	if disco != nil {
		t.Errorf("ent project with no repositories should not emit plugin, got %+v", disco)
	}
}

func TestEntClient_SkipsWhenRepoDiscoNil(t *testing.T) {
	_, disco := contribute(t, nil)
	if disco != nil {
		t.Errorf("no RepoDisco should mean no emission, got %+v", disco)
	}
}
