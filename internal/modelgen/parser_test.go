package modelgen

import (
	"testing"
)

func TestParseArgs_Basic(t *testing.T) {
	def, err := ParseArgs([]string{"Article", "title:string", "body:text", "published:bool"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Name != "Article" {
		t.Errorf("name = %q, want %q", def.Name, "Article")
	}
	if len(def.Fields) != 3 {
		t.Fatalf("fields = %d, want 3", len(def.Fields))
	}

	tests := []struct {
		name  string
		ftype FieldType
	}{
		{"title", FieldString},
		{"body", FieldText},
		{"published", FieldBool},
	}
	for i, tt := range tests {
		if def.Fields[i].Name != tt.name {
			t.Errorf("field[%d].Name = %q, want %q", i, def.Fields[i].Name, tt.name)
		}
		if def.Fields[i].Type != tt.ftype {
			t.Errorf("field[%d].Type = %q, want %q", i, def.Fields[i].Type, tt.ftype)
		}
	}
}

func TestParseArgs_PascalCasesModelName(t *testing.T) {
	def, err := ParseArgs([]string{"blog_post", "title:string"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Name != "BlogPost" {
		t.Errorf("name = %q, want %q", def.Name, "BlogPost")
	}
}

func TestParseArgs_Modifiers(t *testing.T) {
	def, err := ParseArgs([]string{"User", "email:string:uniq", "name:string:index"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !def.Fields[0].Unique {
		t.Error("email should be unique")
	}
	if def.Fields[0].Index {
		t.Error("email should not have index (only uniq)")
	}
	if !def.Fields[1].Index {
		t.Error("name should have index")
	}
}

func TestParseArgs_MultipleModifiers(t *testing.T) {
	def, err := ParseArgs([]string{"User", "email:string:index:uniq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !def.Fields[0].Index || !def.Fields[0].Unique {
		t.Error("email should have both index and uniq")
	}
}

func TestParseArgs_References(t *testing.T) {
	def, err := ParseArgs([]string{"Article", "author:references"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Fields[0].Type != FieldReferences {
		t.Errorf("type = %q, want %q", def.Fields[0].Type, FieldReferences)
	}
	if def.Fields[0].Polymorphic {
		t.Error("should not be polymorphic")
	}
}

func TestParseArgs_PolymorphicReferences(t *testing.T) {
	def, err := ParseArgs([]string{"Comment", "commentable:references{polymorphic}:uniq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := def.Fields[0]
	if f.Type != FieldReferences {
		t.Errorf("type = %q, want %q", f.Type, FieldReferences)
	}
	if !f.Polymorphic {
		t.Error("should be polymorphic")
	}
	if !f.Unique {
		t.Error("should be unique")
	}
}

func TestParseArgs_Optional(t *testing.T) {
	def, err := ParseArgs([]string{"User", "bio:text?"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !def.Fields[0].Optional {
		t.Error("bio should be optional")
	}
}

func TestParseArgs_InvalidType(t *testing.T) {
	_, err := ParseArgs([]string{"User", "name:varchar"})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestParseArgs_InvalidModifier(t *testing.T) {
	_, err := ParseArgs([]string{"User", "name:string:primary"})
	if err == nil {
		t.Fatal("expected error for invalid modifier")
	}
}

func TestParseArgs_PolymorphicOnNonReferences(t *testing.T) {
	_, err := ParseArgs([]string{"User", "name:string{polymorphic}"})
	if err == nil {
		t.Fatal("expected error for polymorphic on string type")
	}
}

func TestParseArgs_NoFields(t *testing.T) {
	def, err := ParseArgs([]string{"User"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(def.Fields) != 0 {
		t.Errorf("fields = %d, want 0", len(def.Fields))
	}
}

func TestParseArgs_Limit(t *testing.T) {
	def, err := ParseArgs([]string{"User", "name:string{444}:index"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := def.Fields[0]
	if f.Limit != 444 {
		t.Errorf("Limit = %d, want 444", f.Limit)
	}
	if !f.Index {
		t.Error("should have index modifier")
	}
}

func TestParseArgs_LimitOnText(t *testing.T) {
	def, err := ParseArgs([]string{"Post", "body:text{5000}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Fields[0].Limit != 5000 {
		t.Errorf("Limit = %d, want 5000", def.Fields[0].Limit)
	}
}

func TestParseArgs_LimitOnInvalidType(t *testing.T) {
	_, err := ParseArgs([]string{"User", "age:int{100}"})
	if err == nil {
		t.Fatal("expected error for limit on int type")
	}
}

func TestParseArgs_LimitZero(t *testing.T) {
	_, err := ParseArgs([]string{"User", "name:string{0}"})
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
}

func TestParseArgs_LimitNegative(t *testing.T) {
	_, err := ParseArgs([]string{"User", "name:string{-1}"})
	if err == nil {
		t.Fatal("expected error for negative limit")
	}
}

func TestParseArgs_EmptyArgs(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Article", "article"},
		{"BlogPost", "blog_post"},
		{"", ""},
		{"URL", "u_r_l"},
	}
	for _, tt := range tests {
		got := SnakeCase(tt.in)
		if got != tt.want {
			t.Errorf("SnakeCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"article", "Article"},
		{"blog_post", "BlogPost"},
		{"Article", "Article"},
		{"", ""},
	}
	for _, tt := range tests {
		got := PascalCase(tt.in)
		if got != tt.want {
			t.Errorf("PascalCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"article", "articles"},
		{"category", "categories"},
		{"bus", "buses"},
		{"", ""},
		{"day", "days"},
	}
	for _, tt := range tests {
		got := Pluralize(tt.in)
		if got != tt.want {
			t.Errorf("Pluralize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestModelDef_HasReferences(t *testing.T) {
	def := &ModelDef{
		Name: "Article",
		Fields: []Field{
			{Name: "title", Type: FieldString},
			{Name: "author", Type: FieldReferences},
		},
	}
	if !def.HasReferences() {
		t.Error("expected HasReferences() to be true")
	}
}

func TestModelDef_HasReferences_PolymorphicOnly(t *testing.T) {
	def := &ModelDef{
		Name: "Comment",
		Fields: []Field{
			{Name: "commentable", Type: FieldReferences, Polymorphic: true},
		},
	}
	if def.HasReferences() {
		t.Error("expected HasReferences() to be false for polymorphic-only")
	}
}

func TestModelDef_HasIndexes(t *testing.T) {
	def := &ModelDef{
		Name: "User",
		Fields: []Field{
			{Name: "email", Type: FieldString, Unique: true},
		},
	}
	if !def.HasIndexes() {
		t.Error("expected HasIndexes() to be true")
	}
}
