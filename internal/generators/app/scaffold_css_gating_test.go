package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCSSGating(t *testing.T) {
	cases := []struct {
		name               string
		data               Data
		wantTailwindDep    bool
		wantTailwindImport bool
		wantSassDep        bool
		wantMuiDep         bool
		wantCarbonDep      bool
	}{
		{name: "bare", data: Data{AppName: "bare", ModulePath: "bare", Renderer: "react", Bundler: "esbuild", Router: "nextjs", CSS: "none", UI: "none", VM: "goja", TestFramework: "none"}},
		{name: "shadcn", data: Data{AppName: "shadcn", ModulePath: "shadcn", Renderer: "react", Bundler: "esbuild", Router: "nextjs", CSS: "tailwind", UI: "shadcn", VM: "goja", TestFramework: "none"}, wantTailwindDep: true, wantTailwindImport: true},
		{name: "mui", data: Data{AppName: "mui", ModulePath: "mui", Renderer: "react", Bundler: "esbuild", Router: "nextjs", CSS: "none", UI: "mui", VM: "goja", TestFramework: "none"}, wantMuiDep: true},
		{name: "carbon", data: Data{AppName: "carbon", ModulePath: "carbon", Renderer: "react", Bundler: "esbuild", Router: "nextjs", CSS: "sass", UI: "carbon", VM: "goja", TestFramework: "none"}, wantSassDep: true, wantCarbonDep: true},
		{name: "explicit-tailwind", data: Data{AppName: "et", ModulePath: "et", Renderer: "react", Bundler: "esbuild", Router: "nextjs", CSS: "tailwind", UI: "none", VM: "goja", TestFramework: "none"}, wantTailwindDep: true, wantTailwindImport: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := Execute(dir, c.data); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			pkg, _ := os.ReadFile(filepath.Join(dir, "web", "package.json"))
			css, _ := os.ReadFile(filepath.Join(dir, "web", "app", "globals.css"))

			if got := strings.Contains(string(pkg), `"tailwindcss":`); got != c.wantTailwindDep {
				t.Errorf("tailwindcss dep: got=%v want=%v\npackage.json:\n%s", got, c.wantTailwindDep, pkg)
			}
			if got := strings.Contains(string(css), `@import "tailwindcss"`); got != c.wantTailwindImport {
				t.Errorf("@import tailwindcss: got=%v want=%v\nglobals.css:\n%s", got, c.wantTailwindImport, css)
			}
			if got := strings.Contains(string(pkg), `"sass":`); got != c.wantSassDep {
				t.Errorf("sass dep: got=%v want=%v", got, c.wantSassDep)
			}
			if got := strings.Contains(string(pkg), `"@mui/material":`); got != c.wantMuiDep {
				t.Errorf("mui dep: got=%v want=%v", got, c.wantMuiDep)
			}
			if got := strings.Contains(string(pkg), `"@carbon/react":`); got != c.wantCarbonDep {
				t.Errorf("carbon dep: got=%v want=%v", got, c.wantCarbonDep)
			}
		})
	}
}
