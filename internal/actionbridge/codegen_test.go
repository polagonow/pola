package actionbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd creates a temporary actions/ package, runs the full codegen
// pipeline, and verifies the generated Go and TypeScript output.
func TestEndToEnd(t *testing.T) {
	// Create a temp directory with a Go module and actions package.
	tmpDir := t.TempDir()

	// Write go.mod
	goMod := `module testapp

go 1.25.0

require github.com/polagonow/pola v0.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create actions directory
	actionsDir := filepath.Join(tmpDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a sample action
	actionSrc := `package actions

type Post struct {
	ID    int    ` + "`json:\"id\"`" + `
	Slug  string ` + "`json:\"slug\"`" + `
	Title string ` + "`json:\"title\"`" + `
}

type Blog struct{}

func (b *Blog) GetPosts() ([]Post, error) {
	return nil, nil
}

func (b *Blog) GetPost(slug string) (*Post, error) {
	return nil, nil
}

func (b *Blog) CreatePost(title string, draft bool) (*Post, error) {
	return nil, nil
}

func (b *Blog) Vars() map[string]any {
	return map[string]any{"appName": "test"}
}
`
	if err := os.WriteFile(filepath.Join(actionsDir, "blog.go"), []byte(actionSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Parse
	result, err := Parse(actionsDir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.StructName != "Blog" {
		t.Errorf("expected Blog, got %s", action.StructName)
	}
	if !action.HasVars {
		t.Error("expected HasVars=true")
	}
	if len(action.Methods) != 3 {
		t.Fatalf("expected 3 methods (excl Vars), got %d", len(action.Methods))
	}

	// Check method details
	methodMap := map[string]MethodDef{}
	for _, m := range action.Methods {
		methodMap[m.JSName] = m
	}

	getPosts := methodMap["getPosts"]
	if !getPosts.HasReturn {
		t.Error("getPosts should have a return value")
	}
	if !strings.Contains(getPosts.ReturnTS, "Post[]") {
		t.Errorf("getPosts return TS: got %q, want Post[]", getPosts.ReturnTS)
	}
	if len(getPosts.Params) != 0 {
		t.Errorf("getPosts params: got %d, want 0", len(getPosts.Params))
	}

	getPost := methodMap["getPost"]
	if len(getPost.Params) != 1 {
		t.Fatalf("getPost params: got %d, want 1", len(getPost.Params))
	}
	if getPost.Params[0].TSType != "string" {
		t.Errorf("getPost param TS: got %q, want string", getPost.Params[0].TSType)
	}
	if getPost.ReturnTS != "Post | null" {
		t.Errorf("getPost return TS: got %q, want Post | null", getPost.ReturnTS)
	}

	createPost := methodMap["createPost"]
	if len(createPost.Params) != 2 {
		t.Fatalf("createPost params: got %d, want 2", len(createPost.Params))
	}
	if createPost.Params[0].TSType != "string" {
		t.Errorf("createPost param[0] TS: got %q, want string", createPost.Params[0].TSType)
	}
	if createPost.Params[1].TSType != "boolean" {
		t.Errorf("createPost param[1] TS: got %q, want boolean", createPost.Params[1].TSType)
	}

	// Test Go generation
	goSrc, err := GenerateGo(result, "github.com/polagonow/pola")
	if err != nil {
		t.Fatalf("GenerateGo: %v", err)
	}
	goStr := string(goSrc)

	if !strings.Contains(goStr, "package actions") {
		t.Error("Go: missing package declaration")
	}
	if !strings.Contains(goStr, `"Blog.getPosts"`) {
		t.Error("Go: missing Blog.getPosts registration")
	}
	if !strings.Contains(goStr, `"Blog.getPost"`) {
		t.Error("Go: missing Blog.getPost registration")
	}
	if !strings.Contains(goStr, "generatedBridge") {
		t.Error("Go: missing generatedBridge struct")
	}
	if !strings.Contains(goStr, "collectVars") {
		t.Error("Go: missing collectVars (Blog has Vars)")
	}
	if !strings.Contains(goStr, "func Plugin()") {
		t.Error("Go: missing Plugin() function")
	}

	// Test TS generation
	tsSrc, err := GenerateTS(result)
	if err != nil {
		t.Fatalf("GenerateTS: %v", err)
	}
	tsStr := string(tsSrc)

	if !strings.Contains(tsStr, "interface Post") {
		t.Error("TS: missing Post interface")
	}
	if !strings.Contains(tsStr, "id: number") {
		t.Error("TS: missing id field in Post")
	}
	if !strings.Contains(tsStr, "slug: string") {
		t.Error("TS: missing slug field in Post")
	}
	if !strings.Contains(tsStr, "getPosts(): Promise<Post[]>") {
		t.Error("TS: missing getPosts method")
	}
	if !strings.Contains(tsStr, "getPost(slug: string): Promise<Post | null>") {
		t.Error("TS: missing getPost method")
	}
	if !strings.Contains(tsStr, "createPost(title: string, draft: boolean): Promise<Post | null>") {
		t.Error("TS: missing createPost method")
	}
	if !strings.Contains(tsStr, "interface BlogActions") {
		t.Error("TS: missing BlogActions interface")
	}
	if !strings.Contains(tsStr, "export const Blog") {
		t.Error("TS: missing Blog export")
	}
	if !strings.Contains(tsStr, `createAction("Blog")`) {
		t.Error("TS: missing createAction call")
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"GetPosts", "getPosts"},
		{"GetPost", "getPost"},
		{"ID", "id"},
		{"HTMLParser", "htmlParser"},
		{"CreatePost", "createPost"},
		{"A", "a"},
		{"", ""},
		{"already", "already"},
	}
	for _, tt := range tests {
		got := CamelCase(tt.in)
		if got != tt.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNoActions(t *testing.T) {
	tmpDir := t.TempDir()
	actionsDir := filepath.Join(tmpDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a Go file with no exported structs that have methods
	src := `package actions

type helper struct{}

func privateFunc() error { return nil }
`
	if err := os.WriteFile(filepath.Join(actionsDir, "empty.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(actionsDir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(result.Actions))
	}
}
