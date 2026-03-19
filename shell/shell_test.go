package shell_test

import (
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/shell"
)

const reactRoot = `<div id="root"></div>`

func TestShellImplementsInterface(t *testing.T) {
	var _ core.HTMLShell = shell.New(reactRoot)
}

func TestShellRenderContainsRootDiv(t *testing.T) {
	s := shell.New(reactRoot)
	out := s.Render(core.ShellParams{
		ClientScript: "/public/assets/client.js",
	})
	if !strings.Contains(out, reactRoot) {
		t.Errorf("Render() output missing %q", reactRoot)
	}
}

func TestShellRenderEmptyInnerHTML(t *testing.T) {
	s := shell.New("")
	out := s.Render(core.ShellParams{ClientScript: "/public/assets/client.js"})
	if !strings.Contains(out, "<body>") {
		t.Error("Render() output missing <body>")
	}
}

func TestShellRenderContainsClientScript(t *testing.T) {
	s := shell.New(reactRoot)
	out := s.Render(core.ShellParams{
		ClientScript: "/public/assets/client-abc123.js",
	})
	if !strings.Contains(out, "/public/assets/client-abc123.js") {
		t.Error("Render() output missing client script URL")
	}
}

func TestShellRenderInlineScripts(t *testing.T) {
	s := shell.New(reactRoot)
	out := s.Render(core.ShellParams{
		ClientScript: "/public/assets/client.js",
		Scripts:      []string{"self.__flight_data=[]"},
	})
	if !strings.Contains(out, "self.__flight_data=[]") {
		t.Error("Render() output missing inline script content")
	}
}

func TestShellRenderImportMap(t *testing.T) {
	s := shell.New(reactRoot)
	out := s.Render(core.ShellParams{
		ClientScript: "/public/assets/client.js",
		ImportURLs:   map[string]string{"./components/Counter": "/public/assets/Counter-abc.js"},
	})
	if !strings.Contains(out, `"importmap"`) {
		t.Error("Render() output missing importmap script tag")
	}
	if !strings.Contains(out, "Counter-abc.js") {
		t.Error("Render() output missing import URL value")
	}
}

func TestShellRenderNoImportMapWhenEmpty(t *testing.T) {
	s := shell.New(reactRoot)
	out := s.Render(core.ShellParams{
		ClientScript: "/public/assets/client.js",
	})
	if strings.Contains(out, `"importmap"`) {
		t.Error("Render() should not emit importmap when ImportURLs is empty")
	}
}

func TestImportMapEmpty(t *testing.T) {
	result := shell.ImportMap(nil)
	if result != "" {
		t.Errorf("ImportMap(nil) = %q, want empty string", result)
	}
}

func TestImportMapNonEmpty(t *testing.T) {
	result := shell.ImportMap(map[string]string{"mod": "/chunk.js"})
	if !strings.Contains(result, `"importmap"`) {
		t.Errorf("ImportMap() missing importmap tag, got: %s", result)
	}
	if !strings.Contains(result, "/chunk.js") {
		t.Errorf("ImportMap() missing URL, got: %s", result)
	}
}
