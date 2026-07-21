// Package dto implements the DTO generator for the Pola CLI. It emits a Go file
// in dto/ with a response DTO plus Create and Update (partial-update) DTOs for a
// resource, wiring the Update DTO to dto.Optional so PATCH requests touch only
// supplied fields.
package dto

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var dtoTmpl = template.Must(
	template.New("dto.go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/dto.go.tmpl"),
)

// DTOGenerator scaffolds request/response DTOs for a resource.
type DTOGenerator struct{}

func init() { generators.Register(&DTOGenerator{}) }

func (g *DTOGenerator) Name() string        { return "dto" }
func (g *DTOGenerator) Description() string  { return "Generate request/response DTOs for a resource" }
func (g *DTOGenerator) AfterHooks() []generators.Hook { return nil }

func (g *DTOGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "dto [Name] [field:type ...]",
		Short: "Generate request/response DTOs for a resource",
		Long: `Generate a Go file in dto/ with a response DTO plus Create and Update
(partial-update) DTOs for the given resource. The Update DTO uses dto.Optional so
a PATCH request touches only the fields the client actually sent.

Field definitions follow the same syntax as the model generator.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Example: `  pola generate dto Product name:string price:float description:text`,
	}
}

// Artifacts implements generators.Destroyer: the file this generator writes for
// the given args, so `pola destroy` (and scaffold cleanup) can find it.
func (g *DTOGenerator) Artifacts(_ *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("name is required")
	}
	def, err := model.ParseArgs(args)
	if err != nil {
		return nil, err
	}
	return []string{filepath.Join(projectDir, "dto", schema.SnakeCase(def.Name)+".go")}, nil
}

func (g *DTOGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}
	def, err := model.ParseArgs(args)
	if err != nil {
		return err
	}

	dtoDir := filepath.Join(projectDir, "dto")
	if err := os.MkdirAll(dtoDir, 0o755); err != nil {
		return fmt.Errorf("create dto dir: %w", err)
	}
	filePath := filepath.Join(dtoDir, schema.SnakeCase(def.Name)+".go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	src, err := render(buildDTOData(def, "dto"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filePath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)
	return nil
}

// render executes the template and gofmt-formats the result, which also
// validates that the generated Go is syntactically well-formed.
func render(data dtoData) ([]byte, error) {
	var buf bytes.Buffer
	if err := dtoTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute dto template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated dto: %w\n%s", err, buf.String())
	}
	return formatted, nil
}

// dtoData is the template payload.
type dtoData struct {
	Package  string
	Name     string
	Imports  []string
	Response []dtoField
	Create   []dtoField
	Update   []dtoField
}

// dtoField is one struct field: a Go name, a Go type, and a struct-tag body
// (without the surrounding backticks).
type dtoField struct {
	GoName string
	GoType string
	Tag    string
}

// buildDTOData turns a parsed model definition into the three DTOs. Hidden
// fields (e.g. a password hash) are kept out of the response but allowed as
// write-only inputs; every input field becomes an Optional on the Update DTO.
func buildDTOData(def *schema.ModelDefinition, pkg string) dtoData {
	d := dtoData{Package: pkg, Name: def.Name}

	needTime := def.HasTimestamps
	needUUID := false
	needJSON := false

	d.Response = append(d.Response, dtoField{GoName: "ID", GoType: def.IDGoType(), Tag: `json:"id"`})

	for _, f := range def.Fields {
		gt := schema.GoType(f.Type)
		switch gt {
		case "time.Time":
			needTime = true
		case "uuid.UUID":
			needUUID = true
		case "json.RawMessage":
			needJSON = true
		}
		name := schema.PascalCase(f.Name)
		jsonName := schema.SnakeCase(f.Name)

		if !f.Hidden {
			d.Response = append(d.Response, dtoField{GoName: name, GoType: gt, Tag: fmt.Sprintf(`json:%q`, jsonName)})
		}

		createTag := fmt.Sprintf(`json:%q`, jsonName)
		if f.Required {
			createTag = fmt.Sprintf(`json:%q validate:"required"`, jsonName)
		}
		d.Create = append(d.Create, dtoField{GoName: name, GoType: gt, Tag: createTag})

		d.Update = append(d.Update, dtoField{
			GoName: name,
			GoType: fmt.Sprintf("poladto.Optional[%s]", gt),
			Tag:    fmt.Sprintf(`json:"%s,omitzero"`, jsonName),
		})
	}

	if def.HasTimestamps {
		d.Response = append(d.Response,
			dtoField{GoName: "CreatedAt", GoType: "time.Time", Tag: `json:"created_at"`},
			dtoField{GoName: "UpdatedAt", GoType: "time.Time", Tag: `json:"updated_at"`},
		)
	}

	if needTime {
		d.Imports = append(d.Imports, `"time"`)
	}
	if needJSON {
		d.Imports = append(d.Imports, `"encoding/json"`)
	}
	if needUUID {
		d.Imports = append(d.Imports, `"github.com/google/uuid"`)
	}
	// The Update DTO always references the framework dto package; alias it so it
	// doesn't clash with the generated file's own `dto` package name.
	d.Imports = append(d.Imports, `poladto "github.com/polagonow/pola/dto"`)

	return d
}
