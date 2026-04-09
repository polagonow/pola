// Package polafile reads and writes Polafile.hcl configuration files.
//
// A Polafile.hcl locks the user's initial project choices (renderer, engine,
// bundler, router, CSS processor, package manager, directory layout) so that
// subsequent CLI commands can pick them up automatically.
//
// The file supports environment-specific nested blocks whose values override
// the top-level block defaults.
package polafile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
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
	UI             string `hcl:"ui,optional"`
	PackageManager string `hcl:"package_manager,optional"`

	App          string `hcl:"app,optional"`
	Actions      string `hcl:"actions,optional"`
	Routes       string `hcl:"routes,optional"`
	Repositories string `hcl:"repositories,optional"`
	Services     string `hcl:"services,optional"`

	CSRF            *CSRF            `hcl:"csrf,block"`
	SecurityHeaders *SecurityHeaders `hcl:"security_headers,block"`
	Cache           *Cache           `hcl:"cache,block"`
	Database        *Database        `hcl:"database,block"`
}

// ---------- CSRF ----------

// CSRFEnvironment holds per-environment CSRF overrides.
type CSRFEnvironment struct {
	Environment string `hcl:"env,label"`
	Enabled     bool   `hcl:"enabled,optional"`
}

// CSRF holds CSRF protection configuration with optional per-environment overrides.
type CSRF struct {
	Enabled bool              `hcl:"enabled,optional"`
	Envs    []CSRFEnvironment `hcl:"env,block"`
}

// CSRFEnabled returns whether CSRF is enabled for the given environment.
// Resolution: env override > base > default (true).
func (pf *Polafile) CSRFEnabled(env string) bool {
	if pf.CSRF == nil {
		return true
	}
	base := pf.CSRF.Enabled
	for _, e := range pf.CSRF.Envs {
		if e.Environment == env {
			return e.Enabled
		}
	}
	return base
}

// ---------- SecurityHeaders ----------

// SecurityHeadersEnvironment holds per-environment security headers overrides.
type SecurityHeadersEnvironment struct {
	Environment string `hcl:"env,label"`
	Enabled     bool   `hcl:"enabled,optional"`
}

// SecurityHeaders holds security headers configuration with optional per-environment overrides.
type SecurityHeaders struct {
	Enabled bool                         `hcl:"enabled,optional"`
	Envs    []SecurityHeadersEnvironment `hcl:"env,block"`
}

// SecurityHeadersEnabled returns whether security headers are enabled for the given environment.
// Resolution: env override > base > default (true).
func (pf *Polafile) SecurityHeadersEnabled(env string) bool {
	if pf.SecurityHeaders == nil {
		return true
	}
	base := pf.SecurityHeaders.Enabled
	for _, e := range pf.SecurityHeaders.Envs {
		if e.Environment == env {
			return e.Enabled
		}
	}
	return base
}

// ---------- Database ----------

// DatabaseEnvironment holds per-environment database overrides.
type DatabaseEnvironment struct {
	Environment string `hcl:"env,label"`
	URL         string `hcl:"url,optional"`
	Host        string `hcl:"host,optional"`
	Port        string `hcl:"port,optional"`
	User        string `hcl:"user,optional"`
	Password    string `hcl:"password,optional"`
	Name        string `hcl:"name,optional"`
	Models      string `hcl:"models,optional"`
	Adapter     string `hcl:"adapter,optional"`
	ORM         string `hcl:"orm,optional"`
}

// Migrations holds migration-specific configuration (shared across environments).
type Migrations struct {
	Directory string `hcl:"directory,optional"`
	Format    string `hcl:"format,optional"`
	DevURL    string `hcl:"dev_url,optional"`
}

// Database holds database configuration with optional per-environment overrides.
type Database struct {
	URL                string                `hcl:"url,optional"`
	Host               string                `hcl:"host,optional"`
	Port               string                `hcl:"port,optional"`
	User               string                `hcl:"user,optional"`
	Password           string                `hcl:"password,optional"`
	Name               string                `hcl:"name,optional"`
	Models             string                `hcl:"models,optional"`
	Adapter            string                `hcl:"adapter,optional"`
	ORM                string                `hcl:"orm,optional"`
	OrmImplementations string                `hcl:"orm_implementations,optional"`
	Migrations         *Migrations           `hcl:"migrations,block"`
	Envs               []DatabaseEnvironment `hcl:"env,block"`
}

// DatabaseForEnv merges the base database config with env-specific overrides.
func (pf *Polafile) DatabaseForEnv(env string) Database {
	if pf.Database == nil {
		return Database{}
	}
	base := Database{
		URL:      pf.Database.URL,
		Host:     pf.Database.Host,
		Port:     pf.Database.Port,
		User:     pf.Database.User,
		Password: pf.Database.Password,
		Name:     pf.Database.Name,
		Models:   pf.Database.Models,
		Adapter:  pf.Database.Adapter,
		ORM:      pf.Database.ORM,
	}
	for _, e := range pf.Database.Envs {
		if e.Environment == env {
			override := Database{
				URL:      e.URL,
				Host:     e.Host,
				Port:     e.Port,
				User:     e.User,
				Password: e.Password,
				Name:     e.Name,
				Models:   e.Models,
				Adapter:  e.Adapter,
				ORM:      e.ORM,
			}
			_ = mergo.Merge(&base, &override, mergo.WithOverride)
			break
		}
	}
	return base
}

// DatabaseModelsDir returns the configured models directory, defaulting to "db/models".
func (pf *Polafile) DatabaseModelsDir() string {
	if pf.Database != nil && pf.Database.Models != "" {
		return pf.Database.Models
	}
	return "db/models"
}

// DatabaseMigrationsDir returns the configured migrations directory, defaulting to "db/migrations".
func (pf *Polafile) DatabaseMigrationsDir() string {
	if pf.Database != nil && pf.Database.Migrations != nil && pf.Database.Migrations.Directory != "" {
		return pf.Database.Migrations.Directory
	}
	return "db/migrations"
}

// DatabaseClientDir returns the base directory for ORM client code, defaulting to "db/client".
func (pf *Polafile) DatabaseClientDir() string {
	if pf.Database != nil && pf.Database.OrmImplementations != "" {
		return pf.Database.OrmImplementations
	}
	return "db/client"
}

// DatabaseEntClientDir returns the directory for the ent-generated client package, defaulting to "db/client/ent".
func (pf *Polafile) DatabaseEntClientDir() string {
	return pf.DatabaseClientDir() + "/ent"
}

// DatabaseMigrationsFormat returns the configured migrations format, defaulting to "sql".
func (pf *Polafile) DatabaseMigrationsFormat() string {
	if pf.Database != nil && pf.Database.Migrations != nil && pf.Database.Migrations.Format != "" {
		return pf.Database.Migrations.Format
	}
	return "sql"
}

// DatabaseURL returns the configured database URL for the given environment.
func (pf *Polafile) DatabaseURL(env string) string {
	merged := pf.DatabaseForEnv(env)
	return merged.URL
}

// DatabaseDevURL returns the configured dev database URL from the migrations block.
func (pf *Polafile) DatabaseDevURL(env string) string {
	if pf.Database != nil && pf.Database.Migrations != nil {
		return pf.Database.Migrations.DevURL
	}
	return ""
}

// DatabaseAdapter returns the configured database adapter for the given environment,
// falling back to base config, then "postgresql".
func (pf *Polafile) DatabaseAdapter(env string) string {
	merged := pf.DatabaseForEnv(env)
	if merged.Adapter != "" {
		return merged.Adapter
	}
	return "postgresql"
}

// DatabaseORM returns the configured ORM, defaulting to "ent".
func (pf *Polafile) DatabaseORM() string {
	if pf.Database != nil && pf.Database.ORM != "" {
		return pf.Database.ORM
	}
	return "ent"
}

// ---------- Cache ----------

// CacheEnvironment holds per-environment cache overrides.
type CacheEnvironment struct {
	Environment string `hcl:"env,label"`
	Enabled     *bool  `hcl:"enabled,optional"`
	Adapter     string `hcl:"adapter,optional"`
	Host        string `hcl:"host,optional"`
	Port        string `hcl:"port,optional"`
	Password    string `hcl:"password,optional"`
	DB          string `hcl:"db,optional"`
}

// Cache holds cache configuration with optional per-environment overrides.
type Cache struct {
	Enabled  bool               `hcl:"enabled,optional"`
	Adapter  string             `hcl:"adapter,optional"`
	Host     string             `hcl:"host,optional"`
	Port     string             `hcl:"port,optional"`
	Password string             `hcl:"password,optional"`
	DB       string             `hcl:"db,optional"`
	Envs     []CacheEnvironment `hcl:"env,block"`
}

// CacheEnabled returns whether caching is enabled for the given environment.
// Resolution: env override > base > default (true).
func (pf *Polafile) CacheEnabled(env string) bool {
	if pf.Cache == nil {
		return true
	}
	base := pf.Cache.Enabled
	for _, e := range pf.Cache.Envs {
		if e.Environment == env && e.Enabled != nil {
			return *e.Enabled
		}
	}
	return base
}

// CacheForEnv merges the base cache config with env-specific overrides.
func (pf *Polafile) CacheForEnv(env string) Cache {
	if pf.Cache == nil {
		return Cache{}
	}
	base := Cache{
		Adapter:  pf.Cache.Adapter,
		Host:     pf.Cache.Host,
		Port:     pf.Cache.Port,
		Password: pf.Cache.Password,
		DB:       pf.Cache.DB,
	}
	for _, e := range pf.Cache.Envs {
		if e.Environment == env {
			override := Cache{
				Adapter:  e.Adapter,
				Host:     e.Host,
				Port:     e.Port,
				Password: e.Password,
				DB:       e.DB,
			}
			_ = mergo.Merge(&base, &override, mergo.WithOverride)
			break
		}
	}
	return base
}

// CacheAdapter returns the configured cache adapter for the given environment,
// falling back to base config, then "memory".
func (pf *Polafile) CacheAdapter(env string) string {
	merged := pf.CacheForEnv(env)
	if merged.Adapter != "" {
		return merged.Adapter
	}
	return "memory"
}

// AppDir returns the configured app directory, defaulting to "web".
func (pf *Polafile) AppDir() string {
	if pf.App != "" {
		return pf.App
	}
	return "web"
}

// RepositoriesDir returns the configured repositories directory, defaulting to "repositories".
func (pf *Polafile) RepositoriesDir() string {
	if pf.Repositories != "" {
		return pf.Repositories
	}
	return "repositories"
}

// ServicesDir returns the configured services directory, defaulting to "services".
func (pf *Polafile) ServicesDir() string {
	if pf.Services != "" {
		return pf.Services
	}
	return "services"
}

// PolaPackage returns the pola framework import path (always DefaultPackage).
func (pf *Polafile) PolaPackage() string {
	return DefaultPackage
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
	setAttr(blockBody, "ui", pf.UI)
	setAttr(blockBody, "package_manager", pf.PackageManager)
	setAttr(blockBody, "app", pf.App)
	setAttr(blockBody, "actions", pf.Actions)
	setAttr(blockBody, "routes", pf.Routes)
	setAttr(blockBody, "repositories", pf.Repositories)
	setAttr(blockBody, "services", pf.Services)

	// CSRF block.
	if pf.CSRF != nil {
		blockBody.AppendNewline()
		csrfBlock := blockBody.AppendNewBlock("csrf", nil)
		csrfBody := csrfBlock.Body()
		csrfBody.SetAttributeValue("enabled", cty.BoolVal(pf.CSRF.Enabled))
		for _, e := range pf.CSRF.Envs {
			envBlock := csrfBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			envBody.SetAttributeValue("enabled", cty.BoolVal(e.Enabled))
		}
	}

	// SecurityHeaders block.
	if pf.SecurityHeaders != nil {
		blockBody.AppendNewline()
		shBlock := blockBody.AppendNewBlock("security_headers", nil)
		shBody := shBlock.Body()
		shBody.SetAttributeValue("enabled", cty.BoolVal(pf.SecurityHeaders.Enabled))
		for _, e := range pf.SecurityHeaders.Envs {
			envBlock := shBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			envBody.SetAttributeValue("enabled", cty.BoolVal(e.Enabled))
		}
	}

	// Cache block.
	if pf.Cache != nil {
		blockBody.AppendNewline()
		cBlock := blockBody.AppendNewBlock("cache", nil)
		cBody := cBlock.Body()
		cBody.SetAttributeValue("enabled", cty.BoolVal(pf.Cache.Enabled))
		setAttr(cBody, "adapter", pf.Cache.Adapter)
		setAttr(cBody, "host", pf.Cache.Host)
		setAttr(cBody, "port", pf.Cache.Port)
		setAttr(cBody, "password", pf.Cache.Password)
		setAttr(cBody, "db", pf.Cache.DB)
		for _, e := range pf.Cache.Envs {
			envBlock := cBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			setAttr(envBody, "adapter", e.Adapter)
			setAttr(envBody, "host", e.Host)
			setAttr(envBody, "port", e.Port)
			setAttr(envBody, "password", e.Password)
			setAttr(envBody, "db", e.DB)
		}
	}

	// Database block.
	if pf.Database != nil {
		blockBody.AppendNewline()
		dbBlock := blockBody.AppendNewBlock("database", nil)
		dbBody := dbBlock.Body()
		setAttr(dbBody, "url", pf.Database.URL)
		setAttr(dbBody, "host", pf.Database.Host)
		setAttr(dbBody, "port", pf.Database.Port)
		setAttr(dbBody, "user", pf.Database.User)
		setAttr(dbBody, "password", pf.Database.Password)
		setAttr(dbBody, "name", pf.Database.Name)
		setAttr(dbBody, "models", pf.Database.Models)
		setAttr(dbBody, "adapter", pf.Database.Adapter)
		setAttr(dbBody, "orm", pf.Database.ORM)
		setAttr(dbBody, "orm_implementations", pf.Database.OrmImplementations)
		if pf.Database.Migrations != nil {
			migBlock := dbBody.AppendNewBlock("migrations", nil)
			migBody := migBlock.Body()
			setAttr(migBody, "directory", pf.Database.Migrations.Directory)
			setAttr(migBody, "format", pf.Database.Migrations.Format)
			setAttr(migBody, "dev_url", pf.Database.Migrations.DevURL)
		}
		for _, e := range pf.Database.Envs {
			envBlock := dbBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			setAttr(envBody, "url", e.URL)
			setAttr(envBody, "host", e.Host)
			setAttr(envBody, "port", e.Port)
			setAttr(envBody, "user", e.User)
			setAttr(envBody, "password", e.Password)
			setAttr(envBody, "name", e.Name)
			setAttr(envBody, "models", e.Models)
			setAttr(envBody, "adapter", e.Adapter)
			setAttr(envBody, "orm", e.ORM)
		}
	}

	path := filepath.Join(dir, Filename)
	if err := os.WriteFile(path, f.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", Filename, err)
	}
	return nil
}
