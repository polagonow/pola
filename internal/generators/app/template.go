// Package app provides project scaffolding for the pola CLI.
package app

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed all:_templates
var templates embed.FS

// Data holds the template variables for scaffolding a new project.
type Data struct {
	AppName       string
	ModulePath    string
	PolaPackage   string // e.g. "github.com/polagonow/pola"
	Renderer      string
	Bundler       string
	Router        string
	Framework     string // HTTP web framework: "std", "gin", "echo", "chi"
	CSS           string
	UI            string
	VM            string
	PolaVersion   string
	PolaLocalPath string // if set, generates a replace directive in go.mod

	// GenerateTests controls whether test scaffolding (vitest/jest config,
	// testhelpers, dev deps) is emitted into the new project. Defaults to true.
	GenerateTests bool
	// TestFramework is "vitest", "jest", or "none". Read by package.json and
	// test config templates to pick the right deps and scripts.
	TestFramework string

	// APIOnly skips renderer/UI template passes (no web/ directory).
	APIOnly bool

	// Populated from uiSpecs[uiKey(UI)] at the start of Execute. The canonical
	// package.json.tmpl ranges over these to merge per-UI extras into the
	// shared React base.
	ExtraDependencies    map[string]string
	ExtraDevDependencies map[string]string
	Scripts              map[string]string

	// NpmOverrides is supplied by the caller (CLI --dependencies flag).
	// NpmVersions is the resolved lookup (defaults merged with overrides) that
	// templates consume via `index .NpmVersions "name"`.
	NpmOverrides map[string]string
	NpmVersions  map[string]string
}

// defaultNpmVersions are the built-in versions for packages that
// `--dependencies` can override. Keys are real npm package names.
var defaultNpmVersions = map[string]string{
	"react":                    "19.2.4",
	"react-dom":                "19.2.4",
	"react-server-dom-webpack": "19.3.0-canary-5e9eedb5-20260312",
	"@types/react":             "^19.2.14",
	"@types/react-dom":         "^19.2.3",
	"tailwindcss":              "^4.2.2",
	"@tailwindcss/cli":         "^4.2.2",
	"sass":                     "^1.89.0",
}

// KnownNpmPackages returns the sorted union of package names that
// `--dependencies` accepts: the canonical defaults plus every package
// declared by any UI overlay (uiSpecs). Used by the CLI for validation.
func KnownNpmPackages() []string {
	seen := make(map[string]struct{}, len(defaultNpmVersions))
	for name := range defaultNpmVersions {
		seen[name] = struct{}{}
	}
	for _, spec := range uiSpecs {
		for name := range spec.Dependencies {
			seen[name] = struct{}{}
		}
		for name := range spec.DevDependencies {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsKnownNpmPackage reports whether name appears in the canonical defaults
// or in any UI overlay. Overrides for packages not present in the selected
// UI (or whose CSS branch is inactive) are silently ignored downstream —
// consistent with how `sass` overrides behave under `--css=tailwind`.
func IsKnownNpmPackage(name string) bool {
	if _, ok := defaultNpmVersions[name]; ok {
		return true
	}
	for _, spec := range uiSpecs {
		if _, ok := spec.Dependencies[name]; ok {
			return true
		}
		if _, ok := spec.DevDependencies[name]; ok {
			return true
		}
	}
	return false
}

// UISpec describes a UI library overlay: its CSS framework requirement
// (drives the auto-detect in cli/new.go) and the extra npm dependencies,
// devDependencies, and scripts to merge into the canonical react
// package.json.tmpl. CSS-driven deps (tailwindcss, sass) are NOT listed
// here — they live as [[ if eq .CSS ]] blocks in the canonical template.
type UISpec struct {
	CSS             string
	Dependencies    map[string]string
	DevDependencies map[string]string
	Scripts         map[string]string
}

// uiSpecs is keyed by --ui flag value. "none" is the bare-React case.
// To add a new UI variant: add an entry here and create a react-{key}/
// overlay directory under _templates/renderers/.
var uiSpecs = map[string]UISpec{
	"none": {},
	"ads":  {},
	"antd": {
		Dependencies: map[string]string{
			"antd":              "^5.25.3",
			"@ant-design/icons": "^5.6.1",
		},
	},
	"carbon": {
		CSS: "sass",
		Dependencies: map[string]string{
			"@carbon/react": "^1.76.0",
		},
	},
	"fluentui": {
		Dependencies: map[string]string{
			"@fluentui/react-components": "^9.56.0",
			"@fluentui/react-icons":      "^2.0.258",
		},
	},
	"mui": {
		Dependencies: map[string]string{
			"@mui/material":       "^7.1.0",
			"@emotion/react":      "^11.14.0",
			"@emotion/styled":     "^11.14.0",
			"@mui/icons-material": "^7.1.0",
		},
	},
	"patternfly": {
		Dependencies: map[string]string{
			"@patternfly/react-core":  "^6.2.0",
			"@patternfly/react-table": "^6.2.0",
			"@patternfly/react-icons": "^6.2.0",
			"@patternfly/patternfly":  "^6.2.0",
		},
	},
	"shadcn": {
		CSS: "tailwind",
		Dependencies: map[string]string{
			"class-variance-authority": "^0.7.1",
			"clsx":                     "^2.1.1",
			"tailwind-merge":           "^3.0.2",
			"lucide-react":             "^0.488.0",
			"tw-animate-css":           "^1.3.4",
			"shadcn":                   "^4.2.0",
			"@radix-ui/react-checkbox": "^1.3.2",
			"@radix-ui/react-label":    "^2.1.6",
			"@radix-ui/react-slot":     "^1.2.3",
		},
	},
	"slds": {
		Dependencies: map[string]string{
			"@salesforce-ux/design-system": "^2.25.5",
		},
		Scripts: map[string]string{
			"postinstall": "cp -r node_modules/@salesforce-ux/design-system/assets public/assets 2>/dev/null || true",
		},
	},
}

// uiKey normalises empty/none to the "none" key for uiSpecs lookup.
func uiKey(ui string) string {
	if ui == "" || ui == "none" {
		return "none"
	}
	return ui
}

// overlayKey returns the suffix of the renderer overlay directory.
// "none" maps to "base" so the bare-React overlay dir is named react-base/.
func overlayKey(ui string) string {
	if ui == "" || ui == "none" {
		return "base"
	}
	return ui
}

// UIRequiresTailwind reports whether the given UI ships its components
// styled with Tailwind. Used by cli/new.go to auto-set --css=tailwind.
func UIRequiresTailwind(_, ui string) bool {
	return uiSpecs[uiKey(ui)].CSS == "tailwind"
}

// UIRequiresSass reports whether the given UI needs Sass for its
// stylesheets. Used by cli/new.go to auto-set --css=sass.
func UIRequiresSass(_, ui string) bool {
	return uiSpecs[uiKey(ui)].CSS == "sass"
}

// Execute renders all embedded templates into targetDir in three passes:
//  1. Shared templates (everything outside renderers/) → targetDir.
//  2. Canonical renderer templates from renderers/<renderer>/ → targetDir/web.
//     This is where the shared tsconfig.json, package.json, and fallback
//     components (error/loading/not-found) live.
//  3. UI overlay from renderers/<renderer>-<overlayKey(UI)>/ → targetDir/web.
//     Overwrites colliding files from pass 2, so UIs can customize layout,
//     page, globals, and optionally any fallback component.
func Execute(targetDir string, data Data) error {
	// Populate per-UI extras for the canonical package.json.tmpl. Copy the
	// maps so per-call NpmOverrides don't mutate the shared uiSpecs globals.
	spec := uiSpecs[uiKey(data.UI)]
	data.ExtraDependencies = copyStringMap(spec.Dependencies)
	data.ExtraDevDependencies = copyStringMap(spec.DevDependencies)
	data.Scripts = spec.Scripts

	// Resolve npm versions: canonical defaults overlaid with caller-supplied
	// overrides. Overrides that match a UI overlay package update the extras
	// maps instead. Unknown-to-this-config overrides are silently ignored.
	data.NpmVersions = make(map[string]string, len(defaultNpmVersions))
	for name, ver := range defaultNpmVersions {
		data.NpmVersions[name] = ver
	}
	for name, ver := range data.NpmOverrides {
		switch {
		case mapHas(data.NpmVersions, name):
			data.NpmVersions[name] = ver
		case mapHas(data.ExtraDependencies, name):
			data.ExtraDependencies[name] = ver
		case mapHas(data.ExtraDevDependencies, name):
			data.ExtraDevDependencies[name] = ver
		}
	}

	// Pass 1: shared templates (skip the renderers/ subtree).
	if err := renderTree(templates, "_templates", targetDir, data, func(rel string) (string, bool) {
		if rel == "renderers" || strings.HasPrefix(rel, "renderers/") || strings.HasPrefix(rel, "renderers\\") {
			return "", false // skip
		}
		if data.APIOnly && (rel == "actions" || strings.HasPrefix(rel, "actions/") || strings.HasPrefix(rel, "actions\\")) {
			return "", false // skip JS bridge sample in API-only mode
		}
		return rel, true
	}); err != nil {
		return fmt.Errorf("shared templates: %w", err)
	}

	if data.APIOnly {
		return nil
	}

	webDir := filepath.Join(targetDir, "web")

	// Pass 2: canonical renderer (e.g. react/) — shared meta + fallback components.
	canonical := filepath.Join("_templates", "renderers", data.Renderer)
	if _, err := fs.Stat(templates, canonical); err != nil {
		return fmt.Errorf("no canonical templates for renderer %q", data.Renderer)
	}
	if err := renderTree(templates, canonical, webDir, data, func(rel string) (string, bool) {
		// Test-framework gating: ship vitest configs only when vitest is the
		// chosen framework (same for jest). Skip both when tests are disabled.
		base := filepath.Base(rel)
		switch base {
		case "vitest.config.ts.tmpl", "vitest.setup.ts.tmpl":
			if !data.GenerateTests || data.TestFramework != "vitest" {
				return "", false
			}
		case "jest.config.ts.tmpl", "jest.setup.ts.tmpl":
			if !data.GenerateTests || data.TestFramework != "jest" {
				return "", false
			}
		}
		return rel, true
	}); err != nil {
		return fmt.Errorf("canonical %s templates: %w", data.Renderer, err)
	}

	// Pass 3: UI overlay (react-base/, react-shadcn/, react-mui/, ...).
	overlayDir := data.Renderer + "-" + overlayKey(data.UI)
	overlayRoot := filepath.Join("_templates", "renderers", overlayDir)
	if _, err := fs.Stat(templates, overlayRoot); err != nil {
		return fmt.Errorf("no overlay templates for %q", overlayDir)
	}
	if err := renderTree(templates, overlayRoot, webDir, data, func(rel string) (string, bool) {
		return rel, true
	}); err != nil {
		return fmt.Errorf("overlay %s templates: %w", overlayDir, err)
	}

	return nil
}

// copyStringMap returns a shallow copy of m, or nil if m is empty.
func copyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// mapHas reports whether m contains key.
func mapHas(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}

// pathFilter returns the output relative path and whether to include the file.
type pathFilter func(rel string) (outRel string, include bool)

// renderTree walks root in the embedded FS and writes rendered files to targetDir.
func renderTree(fsys embed.FS, root, targetDir string, data Data, filter pathFilter) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		outRel, include := filter(rel)
		if !include {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		outPath := filepath.Join(targetDir, outRel)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		// Strip .tmpl extension for output filename.
		outPath = strings.TrimSuffix(outPath, ".tmpl")

		// Restore underscored Go filenames: main_go -> main.go, embed_go -> embed.go
		// (We use _go.tmpl instead of .go.tmpl to prevent the Go toolchain from
		// treating embedded templates as Go source files.)
		base := filepath.Base(outPath)
		if strings.HasSuffix(base, "_go") {
			restored := strings.TrimSuffix(base, "_go") + ".go"
			outPath = filepath.Join(filepath.Dir(outPath), restored)
		}

		// Handle special filenames that can't use their real names in the
		// embedded FS (they'd confuse the Go toolchain).
		switch filepath.Base(outPath) {
		case "gitignore":
			outPath = filepath.Join(filepath.Dir(outPath), ".gitignore")
		case "gomod":
			outPath = filepath.Join(filepath.Dir(outPath), "go.mod")
		}

		// Read template source.
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		// If it's a .tmpl file, execute as template. Otherwise copy as-is.
		var output []byte
		if strings.HasSuffix(path, ".tmpl") {
			tmpl, err := template.New(filepath.Base(path)).Delims("[[", "]]").Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse template %s: %w", path, err)
			}
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				return fmt.Errorf("execute template %s: %w", path, err)
			}
			output = []byte(buf.String())
		} else {
			output = content
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		return os.WriteFile(outPath, output, 0o644)
	})
}
