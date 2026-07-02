// Package repository implements the repository scaffold generator for the Pola CLI.
package repository

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var (
	interfaceTmpl = template.Must(
		template.New("repository_interface.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/repository_interface.tmpl"),
	)
	paginationTmpl = template.Must(
		template.New("pagination_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/pagination_go.tmpl"),
	)
	paginationTestTmpl = template.Must(
		template.New("pagination_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/pagination_test_go.tmpl"),
	)
)

// ormTestTemplate returns the test template paired with the named ORM's
// implementation template. Returns nil if the ORM has no test template.
func ormTestTemplate(orm string) (*template.Template, error) {
	name := orm + "_repository_test.tmpl"
	tmpl, err := template.New(name).Delims("[[", "]]").ParseFS(templates, "_templates/"+name)
	if err != nil {
		return nil, nil // no test template — return nil silently
	}
	return tmpl, nil
}

func ormTemplate(orm string) (*template.Template, error) {
	name := orm + "_repository.tmpl"
	tmpl, err := template.New(name).Delims("[[", "]]").ParseFS(templates, "_templates/"+name)
	if err != nil {
		return nil, fmt.Errorf("unsupported ORM %q for repository generation: %w", orm, err)
	}
	return tmpl, nil
}

// RepositoryGenerator scaffolds repository interfaces and implementations.
type RepositoryGenerator struct{}

func init() {
	generators.Register(&RepositoryGenerator{})
}

func (g *RepositoryGenerator) Name() string { return "repository" }
func (g *RepositoryGenerator) Description() string {
	return "Scaffold a repository interface with ORM implementation"
}
func (g *RepositoryGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("go", "mod", "tidy"),
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *RepositoryGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "repository [Name] [field:type ...]",
		Short: "Scaffold a repository interface with ORM implementation",
		Long: `Generate a repository interface and an ORM-specific implementation
for the given resource. The ORM is read from Polafile.hcl (database.orm).
Field definitions follow the same syntax as the model generator.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"repo"},
		Example: `  pola generate repository User name:string email:string:uniq
  pola generate repository Product name:string price:float`,
	}
}

func repositoryPaths(name, projectDir, repoDir, orm string, generateTests bool) []string {
	snake := schema.SnakeCase(name)
	interfaceDir := filepath.Join(projectDir, repoDir)
	ormDir := filepath.Join(projectDir, repoDir, orm)
	paths := []string{
		filepath.Join(interfaceDir, snake+"_repository.go"),
		filepath.Join(ormDir, snake+"_repository.go"),
	}
	if generateTests {
		paths = append(paths, filepath.Join(ormDir, snake+"_repository_test.go"))
	}
	return paths
}

func (g *RepositoryGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("repository name is required")
	}
	def, err := model.ParseArgs(args)
	if err != nil {
		return nil, err
	}
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}
	repoDir := pf.RepositoriesDir()
	orm := pf.DatabaseORM()
	if orm == "" {
		return nil, fmt.Errorf("ORM not configured in Polafile; cannot determine repository paths")
	}
	genTests := generators.ShouldGenerateTests(cmd, pf.GenerateTests())
	return repositoryPaths(def.Name, projectDir, repoDir, orm, genTests), nil
}

func (g *RepositoryGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	// Load Polafile for repository directory and ORM.
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}

	repoDir := pf.RepositoriesDir()

	// Ensure database block exists; prompt interactively for missing ORM.
	if pf.Database == nil {
		pf.Database = &polafile.Database{}
	}
	dirty := false
	if pf.Database.ORM == "" {
		orm, err := promptSelect("ORM:", []string{"beego", "ent", "gorm"})
		if err != nil {
			return err
		}
		pf.Database.ORM = orm
		dirty = true
	}
	if dirty {
		if err := polafile.Save(projectDir, pf); err != nil {
			return fmt.Errorf("save Polafile: %w", err)
		}
		fmt.Println("Saved database configuration to Polafile.hcl.")
	}

	orm := pf.DatabaseORM()
	ormTmpl, err := ormTemplate(orm)
	if err != nil {
		return err
	}

	def, err := model.ParseArgs(args)
	if err != nil {
		return err
	}

	data := buildData(def, modulePath)
	data.PolaPackage = pf.PolaPackage()
	data.EntClientDir = pf.DatabaseEntClientDir()

	// Ensure repository directory exists.
	interfaceDir := filepath.Join(projectDir, repoDir)
	if err := os.MkdirAll(interfaceDir, 0o755); err != nil {
		return fmt.Errorf("create repository dir: %w", err)
	}

	// Generate shared pagination.go once (skip if it already exists).
	paginationPath := filepath.Join(interfaceDir, "pagination.go")
	if _, err := os.Stat(paginationPath); os.IsNotExist(err) {
		var pbuf strings.Builder
		if err := paginationTmpl.Execute(&pbuf, nil); err != nil {
			return fmt.Errorf("execute pagination template: %w", err)
		}
		if err := os.WriteFile(paginationPath, []byte(pbuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", paginationPath, err)
		}
		fmt.Printf("Created %s\n", paginationPath)
	}

	// Generate shared pagination_test.go once.
	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		paginationTestPath := filepath.Join(interfaceDir, "pagination_test.go")
		if _, err := os.Stat(paginationTestPath); os.IsNotExist(err) {
			var pbuf strings.Builder
			if err := paginationTestTmpl.Execute(&pbuf, nil); err != nil {
				return fmt.Errorf("execute pagination test template: %w", err)
			}
			if err := os.WriteFile(paginationTestPath, []byte(pbuf.String()), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", paginationTestPath, err)
			}
			fmt.Printf("Created %s\n", paginationTestPath)
		}
	}

	// Generate interface file.
	interfacePath := filepath.Join(interfaceDir, data.SnakeName+"_repository.go")
	if err := generators.CheckCollision(cmd, interfacePath); err != nil {
		return err
	}
	if err := writeTemplate(interfaceTmpl, interfacePath, data); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", interfacePath)

	// Generate ORM-specific implementation.
	ormDir := filepath.Join(projectDir, repoDir, orm)
	if err := os.MkdirAll(ormDir, 0o755); err != nil {
		return fmt.Errorf("create %s/%s dir: %w", repoDir, orm, err)
	}
	ormPath := filepath.Join(ormDir, data.SnakeName+"_repository.go")
	if err := generators.CheckCollision(cmd, ormPath); err != nil {
		return err
	}
	if err := writeTemplate(ormTmpl, ormPath, data); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", ormPath)

	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testTmpl, _ := ormTestTemplate(orm)
		if testTmpl != nil {
			ormTestPath := filepath.Join(ormDir, data.SnakeName+"_repository_test.go")
			if err := generators.CheckCollision(cmd, ormTestPath); err != nil {
				return err
			}
			if err := writeTemplate(testTmpl, ormTestPath, data); err != nil {
				return err
			}
			fmt.Printf("Created %s\n", ormTestPath)
		}
	}

	return generators.RunAfterHooks(g, projectDir)
}

// promptSelect presents an interactive selection prompt and returns the chosen option.
func promptSelect(message string, options []string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}
	return answer, nil
}

type repoData struct {
	Name         string
	LowerName    string
	SnakeName    string
	PluralSnake  string
	EntPackage   string // all-lowercase name used by ent for sub-packages (e.g. "sampleentity")
	EntClientDir string // directory for ent-generated client (e.g. "db/ent")
	IDGoType     string // "uint" or "string"
	Fields       []repoField
	Imports      []string
	ModulePath   string
	PolaPackage  string
}

type repoField struct {
	Name     string
	JSONName string
	GoType   string
	ValidTag string
}

func buildData(def *schema.ModelDefinition, modulePath string) repoData {
	idGoType := "uint"
	if def.HasUUIDPrimaryKey() {
		idGoType = "string"
	}

	data := repoData{
		Name:        def.Name,
		LowerName:   strings.ToLower(def.Name[:1]) + def.Name[1:],
		SnakeName:   schema.SnakeCase(def.Name),
		PluralSnake: schema.SnakeCase(schema.Pluralize(def.Name)),
		EntPackage:  strings.ToLower(def.Name),
		IDGoType:    idGoType,
		ModulePath:  modulePath,
	}

	imports := map[string]bool{}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences {
			if f.RefModel == "StorageBlob" {
				// Include blob FK field (e.g. AvatarID uint) so routes can set it.
				data.Fields = append(data.Fields, repoField{
					Name:     schema.PascalCase(f.Name) + "ID",
					JSONName: schema.SnakeCase(f.Name) + "_id",
					GoType:   "uint",
					ValidTag: "omitempty",
				})
			}
			continue
		}
		goType := schema.GoType(f.Type)
		data.Fields = append(data.Fields, repoField{
			Name:     schema.PascalCase(f.Name),
			JSONName: schema.SnakeCase(f.Name),
			GoType:   goType,
			ValidTag: validTagForField(f),
		})
		switch f.Type {
		case schema.FieldTime:
			imports[`"time"`] = true
		case schema.FieldUUID:
			imports[`"github.com/google/uuid"`] = true
		case schema.FieldJSON:
			imports[`"encoding/json"`] = true
		}
	}

	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	return data
}

// validatorTag maps schema field types to go-playground/validator tag
// fragments. Types without a built-in go-playground tag (e.g. Port) are omitted
// and validated only by required/omitempty.
var validatorTag = map[schema.FieldType]string{
	schema.FieldUUID:         "uuid4",
	schema.FieldEmail:        "email",
	schema.FieldURL:          "url",
	schema.FieldIP:           "ip",
	schema.FieldIPv4:         "ipv4",
	schema.FieldIPv6:         "ipv6",
	schema.FieldMAC:          "mac",
	schema.FieldAlpha:        "alpha",
	schema.FieldNumeric:      "numeric",
	schema.FieldAlphanumeric: "alphanum",
	schema.FieldHexColor:     "hexcolor",
	schema.FieldCreditCard:   "credit_card",
	schema.FieldBase64:       "base64",
	schema.FieldLatitude:     "latitude",
	schema.FieldLongitude:    "longitude",
	schema.FieldSemver:       "semver",
}

func validTagForField(f schema.Field) string {
	if f.Type == schema.FieldBool {
		return "-"
	}
	base := "required"
	if f.Optional {
		base = "omitempty"
	}
	if tag, ok := validatorTag[f.Type]; ok {
		return base + "," + tag
	}
	return base
}

func writeTemplate(tmpl *template.Template, path string, data repoData) error {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
