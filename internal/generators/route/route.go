// Package route implements the route scaffold generator for the Pola CLI.
package route

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var (
	routeTmpl = template.Must(
		template.New("route_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/route_go.tmpl"),
	)
	routeServiceTmpl = template.Must(
		template.New("route_service_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/route_service_go.tmpl"),
	)
	routeServiceUploadTmpl = template.Must(
		template.New("route_service_upload_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/route_service_upload_go.tmpl"),
	)
)

// fileField describes a file upload field for the upload route template.
type fileField struct {
	Name       string // snake_case: "avatar"
	PascalName string // PascalCase: "Avatar"
}

// validHTTPMethods is the set of accepted HTTP methods.
var validHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	"HEAD": true, "OPTIONS": true, "CONNECT": true, "TRACE": true,
}

// RouteGenerator scaffolds new route handlers in the routes/ directory.
type RouteGenerator struct{}

func init() {
	generators.Register(&RouteGenerator{})
}

func (g *RouteGenerator) Name() string                  { return "route" }
func (g *RouteGenerator) Description() string           { return "Scaffold a new route handler" }
func (g *RouteGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{generators.CmdHook("gofmt", "-w", ".")}
}

func (g *RouteGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [Name] [methods...]",
		Short: "Scaffold a new route handler",
		Long: `Create a new route file in the routes/ directory with HTTP method stubs.

Methods can be passed as separate arguments or comma-separated.
If no methods are provided, defaults to GET.

Use --service=Name to wire the route to a generated service via DI.`,
		Args: cobra.MinimumNArgs(1),
		RunE: g.run,
		Example: `  pola generate route Posts
  pola generate route Posts GET,POST
  pola generate route Posts/Comments GET POST DELETE
  pola generate route Posts GET,POST,DELETE --service=Post`,
	}
	cmd.Flags().String("service", "", "wire route handlers to the named service via DI")
	cmd.Flags().String("id-type", "uint", "Go type for the entity ID (uint or string)")
	cmd.Flags().MarkHidden("id-type")
	cmd.Flags().Bool("has-file-upload", false, "generate multipart file upload handling")
	cmd.Flags().MarkHidden("has-file-upload")
	cmd.Flags().String("file-fields", "", "comma-separated snake_case file field names")
	cmd.Flags().MarkHidden("file-fields")
	return cmd
}

func (g *RouteGenerator) run(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Split on "/" for nested routes: Posts/Comments → ["posts", "comments"].
	// Each segment is pluralized: Product → products, Category → categories.
	segments := strings.Split(name, "/")
	for i, s := range segments {
		segments[i] = schema.Pluralize(strings.ToLower(s))
	}
	pkgName := segments[len(segments)-1]

	// Parse and validate HTTP methods from positional args.
	methods, err := parseActions(args[1:])
	if err != nil {
		return err
	}

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(append([]string{projectDir, "routes"}, segments...)...)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create route dir: %w", err)
	}

	filePath := filepath.Join(targetDir, "route.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	routePath := "/" + strings.Join(segments, "/")

	serviceName, _ := cmd.Flags().GetString("service")

	var buf strings.Builder
	if serviceName != "" {
		// Capitalize first letter.
		if serviceName[0] >= 'a' && serviceName[0] <= 'z' {
			serviceName = string(serviceName[0]-32) + serviceName[1:]
		}

		modulePath, err := project.ModulePath(projectDir)
		if err != nil {
			return fmt.Errorf("read module path: %w", err)
		}

		idType, _ := cmd.Flags().GetString("id-type")
		if idType == "" {
			idType = "uint"
		}

		hasFileUpload, _ := cmd.Flags().GetBool("has-file-upload")
		fileFieldsStr, _ := cmd.Flags().GetString("file-fields")

		if hasFileUpload && fileFieldsStr != "" {
			var fields []fileField
			for _, name := range strings.Split(fileFieldsStr, ",") {
				name = strings.TrimSpace(name)
				if name != "" {
					fields = append(fields, fileField{
						Name:       name,
						PascalName: schema.PascalCase(name),
					})
				}
			}
			if err := routeServiceUploadTmpl.Execute(&buf, struct {
				Package     string
				RoutePath   string
				Methods     []string
				ServiceName string
				IDGoType    string
				ModulePath  string
				FileFields  []fileField
			}{
				Package:     pkgName,
				RoutePath:   routePath,
				Methods:     methods,
				ServiceName: serviceName,
				IDGoType:    idType,
				ModulePath:  modulePath,
				FileFields:  fields,
			}); err != nil {
				return fmt.Errorf("execute route service upload template: %w", err)
			}
		} else if err := routeServiceTmpl.Execute(&buf, struct {
			Package     string
			RoutePath   string
			Methods     []string
			ServiceName string
			IDGoType    string
			ModulePath  string
		}{
			Package:     pkgName,
			RoutePath:   routePath,
			Methods:     methods,
			ServiceName: serviceName,
			IDGoType:    idType,
			ModulePath:  modulePath,
		}); err != nil {
			return fmt.Errorf("execute route service template: %w", err)
		}
	} else {
		if err := routeTmpl.Execute(&buf, struct {
			Package   string
			RoutePath string
			Methods   []string
		}{
			Package:   pkgName,
			RoutePath: routePath,
			Methods:   methods,
		}); err != nil {
			return fmt.Errorf("execute route template: %w", err)
		}
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	return generators.RunAfterHooks(g, projectDir)
}

// parseActions validates HTTP methods from positional arguments.
// Each argument may be a single method or comma-separated (e.g. "GET,POST").
// Returns ["GET"] if no methods are provided.
func parseActions(args []string) ([]string, error) {
	seen := make(map[string]bool)
	var methods []string
	for _, arg := range args {
		// Skip flag-like arguments so --service=X doesn't get parsed as a method.
		if strings.HasPrefix(arg, "--") {
			continue
		}
		for _, p := range strings.Split(arg, ",") {
			m := strings.TrimSpace(strings.ToUpper(p))
			if m == "" {
				continue
			}
			if !validHTTPMethods[m] {
				return nil, fmt.Errorf("unknown HTTP method %q; valid methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, CONNECT, TRACE", m)
			}
			if seen[m] {
				continue
			}
			seen[m] = true
			methods = append(methods, m)
		}
	}
	if len(methods) == 0 {
		methods = []string{"GET"}
	}
	return methods, nil
}
