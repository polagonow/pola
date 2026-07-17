package route

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"text/template"
)

// allMethods exercises every branch in the templates.
var allMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

func render(t *testing.T, tmpl *template.Template, data any) string {
	t.Helper()
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute %s: %v", tmpl.Name(), err)
	}
	src := buf.String()
	if _, err := parser.ParseFile(token.NewFileSet(), "out.go", src, parser.AllErrors); err != nil {
		t.Fatalf("%s produced invalid Go: %v\n%s", tmpl.Name(), err, src)
	}
	return src
}

func TestRouteTemplatesRender(t *testing.T) {
	type basic struct {
		Package      string
		RoutePath    string
		ExplicitPath string
		Methods      []string
	}
	type withFunc struct {
		Package   string
		RoutePath string
		Methods   []string
		Func      bool
	}
	type svc struct {
		Package      string
		RoutePath    string
		ExplicitPath string
		Methods      []string
		ServiceName  string
		IDGoType     string
		ModulePath   string
	}
	type upload struct {
		Package      string
		RoutePath    string
		ExplicitPath string
		Methods      []string
		ServiceName  string
		IDGoType     string
		ModulePath   string
		FileFields   []fileField
	}

	b := basic{Package: "posts", RoutePath: "/posts", Methods: allMethods}
	render(t, routeTmpl, b)
	render(t, routeFuncTmpl, b)
	// Exercise the --path branch (emits a Path() override).
	render(t, routeTmpl, basic{Package: "user", RoutePath: "/api/user", ExplicitPath: "/api/user", Methods: allMethods})
	render(t, routeTestTmpl, withFunc{Package: "posts", RoutePath: "/posts", Methods: allMethods, Func: false})
	render(t, routeTestTmpl, withFunc{Package: "posts", RoutePath: "/posts", Methods: allMethods, Func: true})

	for _, idType := range []string{"uint", "string"} {
		s := svc{Package: "posts", RoutePath: "/posts", Methods: allMethods, ServiceName: "Post", IDGoType: idType, ModulePath: "example.com/app"}
		render(t, routeServiceTmpl, s)
		render(t, routeServiceTestTmpl, s)
		render(t, routeServiceUploadTmpl, upload{
			Package: "posts", RoutePath: "/posts", Methods: allMethods, ServiceName: "Post",
			IDGoType: idType, ModulePath: "example.com/app",
			FileFields: []fileField{{Name: "avatar", PascalName: "Avatar"}},
		})
		render(t, routeServiceUploadTestTmpl, s)
	}
}
