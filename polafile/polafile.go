// Package polafile reads and writes Polafile.hcl configuration files.
//
// A Polafile.hcl locks the user's initial project choices (renderer, engine,
// bundler, router, CSS processor, package manager, directory layout) so that
// subsequent CLI commands can pick them up automatically.
//
// The file supports environment-specific blocks (development, production)
// whose values override the top-level pola block defaults.
package polafile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ParseVersioned splits a "name@version" string into its parts.
// If no "@" is present, version is returned as empty.
//
//	ParseVersioned("tailwind@^4.0.0") => "tailwind", "^4.0.0"
//	ParseVersioned("react")           => "react", ""
func ParseVersioned(s string) (name, version string) {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// FormatVersioned joins a name and version into "name@version".
// If version is empty, only the name is returned.
func FormatVersioned(name, version string) string {
	if version == "" {
		return name
	}
	return name + "@" + version
}

// Filename is the expected config file name.
const Filename = "Polafile.hcl"

// polafileSchema is the top-level HCL structure containing a pola block.
type polafileSchema struct {
	Pola Polafile `hcl:"pola,block"`
}

// DefaultPackage is the default Go import path for the pola framework.
const DefaultPackage = "github.com/polagonow/pola"

// Polafile represents the contents of a Polafile.hcl.
type Polafile struct {
	Package        string `hcl:"package,optional"`
	Version        string `hcl:"version,optional"`
	Renderer       string `hcl:"renderer,optional"`
	Engine         string `hcl:"engine,optional"`
	Bundler        string `hcl:"bundler,optional"`
	Router         string `hcl:"router,optional"`
	CSS            string `hcl:"css,optional"`
	Cache          string `hcl:"cache,optional"`
	PackageManager string `hcl:"package_manager,optional"`

	AppDir     string `hcl:"app_dir,optional"`
	ActionsDir string `hcl:"actions_dir,optional"`
	RoutesDir  string `hcl:"routes_dir,optional"`

	Development *Environment `hcl:"development,block"`
	Production  *Environment `hcl:"production,block"`
}

// PolaPackage returns the configured pola framework import path,
// falling back to DefaultPackage if not set.
func (pf *Polafile) PolaPackage() string {
	if pf.Package != "" {
		return pf.Package
	}
	return DefaultPackage
}

// Environment holds per-environment overrides.
type Environment struct {
	Renderer       string `hcl:"renderer,optional"`
	Engine         string `hcl:"engine,optional"`
	Bundler        string `hcl:"bundler,optional"`
	Router         string `hcl:"router,optional"`
	CSS            string `hcl:"css,optional"`
	Cache          string `hcl:"cache,optional"`
	PackageManager string `hcl:"package_manager,optional"`
}

// ForEnv returns a merged Polafile with the given environment's overrides
// applied on top of the base values.
func (pf *Polafile) ForEnv(env string) Polafile {
	merged := *pf

	var override *Environment
	switch env {
	case "development":
		override = pf.Development
	case "production":
		override = pf.Production
	}
	if override == nil {
		return merged
	}

	if override.Renderer != "" {
		merged.Renderer = override.Renderer
	}
	if override.Engine != "" {
		merged.Engine = override.Engine
	}
	if override.Bundler != "" {
		merged.Bundler = override.Bundler
	}
	if override.Router != "" {
		merged.Router = override.Router
	}
	if override.CSS != "" {
		merged.CSS = override.CSS
	}
	if override.Cache != "" {
		merged.Cache = override.Cache
	}
	if override.PackageManager != "" {
		merged.PackageManager = override.PackageManager
	}

	return merged
}

// Load reads and parses a Polafile.hcl from the given directory.
// Returns nil, nil if no Polafile.hcl exists.
func Load(dir string) (*Polafile, error) {
	path := filepath.Join(dir, Filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", Filename, err)
	}

	var schema polafileSchema
	if err := hclsimple.Decode(Filename, data, nil, &schema); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Filename, err)
	}
	return &schema.Pola, nil
}

// Save writes a Polafile.hcl to the given directory.
func Save(dir string, pf *Polafile) error {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	block := body.AppendNewBlock("pola", nil)
	blockBody := block.Body()

	setAttr := func(b *hclwrite.Body, key, val string) {
		if val != "" {
			b.SetAttributeValue(key, cty.StringVal(val))
		}
	}

	setAttr(blockBody, "package", pf.Package)
	setAttr(blockBody, "version", pf.Version)
	setAttr(blockBody, "renderer", pf.Renderer)
	setAttr(blockBody, "engine", pf.Engine)
	setAttr(blockBody, "bundler", pf.Bundler)
	setAttr(blockBody, "router", pf.Router)
	setAttr(blockBody, "css", pf.CSS)
	setAttr(blockBody, "cache", pf.Cache)
	setAttr(blockBody, "package_manager", pf.PackageManager)
	setAttr(blockBody, "app_dir", pf.AppDir)
	setAttr(blockBody, "actions_dir", pf.ActionsDir)
	setAttr(blockBody, "routes_dir", pf.RoutesDir)

	writeEnvBlock := func(name string, env *Environment) {
		if env == nil {
			return
		}
		blockBody.AppendNewline()
		envBlock := blockBody.AppendNewBlock(name, nil)
		envBody := envBlock.Body()
		setAttr(envBody, "renderer", env.Renderer)
		setAttr(envBody, "engine", env.Engine)
		setAttr(envBody, "bundler", env.Bundler)
		setAttr(envBody, "router", env.Router)
		setAttr(envBody, "css", env.CSS)
		setAttr(envBody, "cache", env.Cache)
		setAttr(envBody, "package_manager", env.PackageManager)
	}

	writeEnvBlock("development", pf.Development)
	writeEnvBlock("production", pf.Production)

	path := filepath.Join(dir, Filename)
	if err := os.WriteFile(path, f.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", Filename, err)
	}
	return nil
}
