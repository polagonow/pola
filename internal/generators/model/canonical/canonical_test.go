package canonical

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

// TestRoundTrip proves the canonical emitter and schema.ParseModelFile are
// inverses: Generate(def) then ParseModelFile must reproduce def. This is the
// guarantee that editing a struct and generating from the CLI converge on the
// same IR.
func TestRoundTrip(t *testing.T) {
	cases := []*schema.ModelDefinition{
		{
			Name:          "Todo",
			HasTimestamps: true,
			SoftDeletes:   true,
			Fields: []schema.Field{
				{Name: "title", Type: schema.FieldString, Limit: 255, Index: true},
				{Name: "slug", Type: schema.FieldString, Limit: 120, Unique: true},
				{Name: "note", Type: schema.FieldText, Optional: true},
				{Name: "completed", Type: schema.FieldBool},
				{Name: "email", Type: schema.FieldEmail, Unique: true},
				{Name: "count", Type: schema.FieldInt},
			},
		},
		{
			Name:   "Session",
			IDType: schema.FieldUUID,
			Fields: []schema.Field{
				{Name: "token", Type: schema.FieldString, Limit: 255, Unique: true},
			},
		},
		{
			Name:          "Comment",
			HasTimestamps: true,
			Fields: []schema.Field{
				{Name: "body", Type: schema.FieldText},
				{Name: "author", Type: schema.FieldReferences, RefModel: "User"},
			},
		},
	}
	for _, def := range cases {
		dir := filepath.Join(t.TempDir(), "models")
		if err := Generate(def, dir); err != nil {
			t.Fatalf("Generate(%s): %v", def.Name, err)
		}
		got, err := schema.ParseModelFile(filepath.Join(dir, schema.SnakeCase(def.Name)+".go"))
		if err != nil {
			t.Fatalf("ParseModelFile(%s): %v", def.Name, err)
		}
		if !reflect.DeepEqual(got, def) {
			t.Errorf("round-trip mismatch for %s:\n got  %+v\n want %+v", def.Name, got, def)
		}
	}
}
