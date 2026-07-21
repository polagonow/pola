package dto

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/model/schema"
)

func parseValid(t *testing.T, src string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "dto.go", []byte(src), parser.AllErrors); err != nil {
		t.Fatalf("generated code is not valid Go: %v\n%s", err, src)
	}
}

func render_(t *testing.T, args ...string) string {
	t.Helper()
	def, err := model.ParseArgs(args)
	if err != nil {
		t.Fatalf("ParseArgs(%v): %v", args, err)
	}
	src, err := render(buildDTOData(def, "dto"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// gofmt already ran inside render; double-check it parses as valid Go.
	if _, err := parser.ParseFile(token.NewFileSet(), "dto.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated code is not valid Go: %v\n%s", err, src)
	}
	return string(src)
}

func TestGeneratesThreeDTOs(t *testing.T) {
	s := render_(t, "Product", "name:string", "price:float", "count:int")

	for _, want := range []string{
		"type Product struct",
		"type CreateProduct struct",
		"type UpdateProduct struct",
		`json:"id"`,                // auto-increment PK on the response DTO
		"poladto.Optional[string]", // Update wraps inputs in Optional
		"poladto.Optional[float64]",
		`json:"name,omitzero"`, // omitzero so absent PATCH fields disappear
		`poladto "github.com/polagonow/pola/dto"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("generated dto missing %q\n---\n%s", want, s)
		}
	}
}

func TestRequiredAddsValidateTag(t *testing.T) {
	s := render_(t, "Article", "title:string!") // "!" marks required
	if !strings.Contains(s, `validate:"required"`) {
		t.Errorf("required field should carry a validate tag\n%s", s)
	}
}

func TestTimestampsAndImports(t *testing.T) {
	// A model with timestamps adds CreatedAt/UpdatedAt to the response DTO and
	// pulls in the time import.
	def := &schema.ModelDefinition{
		Name:          "Event",
		Fields:        []schema.Field{{Name: "name", Type: schema.FieldString}},
		HasTimestamps: true,
	}
	src, err := render(buildDTOData(def, "dto"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	parseValid(t, string(src))

	s := string(src)
	if !strings.Contains(s, "CreatedAt time.Time") || !strings.Contains(s, `"time"`) {
		t.Errorf("timestamps not reflected in response/imports\n%s", s)
	}
}
