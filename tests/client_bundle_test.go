package tests

import (
	"os"
	"strings"
	"testing"
)

func TestClientBundle_Served(t *testing.T) {
	app := requireApp(t)
	if app.Artifacts().Output.ClientEntryURL == "" {
		t.Fatal("ClientEntryScript is empty")
	}
	if !strings.HasPrefix(app.Artifacts().Output.ClientEntryURL, "/public/") {
		t.Errorf("ClientEntryScript should start with /public/, got %q", app.Artifacts().Output.ClientEntryURL)
	}
}

func TestClientBundle_FileExists(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.Artifacts().Output.ClientEntryURL, "/public/")
	path := "../ui/apps/blog/public/" + rel
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("client bundle not found at %q: %v", path, err)
	}
	if info.Size() < 10_000 {
		t.Errorf("client bundle suspiciously small (%d bytes)", info.Size())
	}
}

func TestClientBundle_NoWebpackRequire(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.Artifacts().Output.ClientEntryURL, "/public/")
	data, err := os.ReadFile("../ui/apps/blog/public/" + rel)
	if err != nil {
		t.Fatalf("read client bundle: %v", err)
	}
	if strings.Contains(string(data), "__webpack_require__") {
		t.Error("client bundle contains __webpack_require__")
	}
}
