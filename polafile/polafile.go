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
	Storage         *StorageConfig   `hcl:"storage,block"`
	Mailer          *Mailer          `hcl:"mailer,block"`
	ImageProcessing *ImageProcessing `hcl:"image_processing,block"`
	MCP             *MCP             `hcl:"mcp,block"`
	Testing         *Testing         `hcl:"testing,block"`
}

// ---------- Testing ----------

// Testing holds test-generation configuration. When unset, generators default
// to producing tests (Rails-style) and use the Vitest framework for TS tests.
type Testing struct {
	// GenerateTests controls whether `pola generate <thing>` also emits test
	// files. Use *bool so we can distinguish "unset" from "explicitly false".
	GenerateTests *bool `hcl:"generate_tests,optional"`
	// Framework is the JavaScript/TypeScript test framework: "vitest" or "jest".
	Framework string `hcl:"framework,optional"`
}

// GenerateTests reports whether test files should be emitted by generators.
// Defaults to true when unset.
func (pf *Polafile) GenerateTests() bool {
	if pf == nil || pf.Testing == nil || pf.Testing.GenerateTests == nil {
		return true
	}
	return *pf.Testing.GenerateTests
}

// TestFramework returns the configured TS test framework, defaulting to "vitest".
func (pf *Polafile) TestFramework() string {
	if pf == nil || pf.Testing == nil || pf.Testing.Framework == "" {
		return "vitest"
	}
	return pf.Testing.Framework
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

// ---------- Storage ----------

// StorageEnvironment holds per-environment storage overrides.
type StorageEnvironment struct {
	Environment string `hcl:"env,label"`
	Driver      string `hcl:"driver,optional"`
	Root        string `hcl:"root,optional"`
	ConfigPath  string `hcl:"config_path,optional"`
}

// StorageConfig holds file storage configuration with optional per-environment overrides.
// Driver is "fs" (local filesystem) or "rclone" (any rclone-supported backend).
// Root is the storage path: a local directory for fs, or an rclone remote path
// (e.g. "myremote:bucket/path") for rclone.
// ConfigPath is the path to the rclone config file (optional, defaults to rclone's default).
type StorageConfig struct {
	Driver     string               `hcl:"driver,optional"`
	Root       string               `hcl:"root,optional"`
	ConfigPath string               `hcl:"config_path,optional"`
	Envs       []StorageEnvironment `hcl:"env,block"`
}

// StorageForEnv merges the base storage config with env-specific overrides.
func (pf *Polafile) StorageForEnv(env string) StorageConfig {
	if pf.Storage == nil {
		return StorageConfig{}
	}
	base := StorageConfig{
		Driver:     pf.Storage.Driver,
		Root:       pf.Storage.Root,
		ConfigPath: pf.Storage.ConfigPath,
	}
	for _, e := range pf.Storage.Envs {
		if e.Environment == env {
			override := StorageConfig{
				Driver:     e.Driver,
				Root:       e.Root,
				ConfigPath: e.ConfigPath,
			}
			_ = mergo.Merge(&base, &override, mergo.WithOverride)
			break
		}
	}
	return base
}

// StorageDriver returns the configured storage driver, defaulting to "fs".
func (pf *Polafile) StorageDriver() string {
	if pf.Storage != nil && pf.Storage.Driver != "" {
		return pf.Storage.Driver
	}
	return "fs"
}

// StorageRoot returns the configured storage root directory, defaulting to "uploads".
func (pf *Polafile) StorageRoot() string {
	if pf.Storage != nil && pf.Storage.Root != "" {
		return pf.Storage.Root
	}
	return "uploads"
}

// StorageConfigPath returns the configured rclone config path for the given environment.
func (pf *Polafile) StorageConfigPath(env string) string {
	merged := pf.StorageForEnv(env)
	return merged.ConfigPath
}

// AppDir returns the configured app directory, defaulting to "web".
func (pf *Polafile) AppDir() string {
	if pf.App != "" {
		return pf.App
	}
	return "web"
}

// ---------- Mailer ----------

// MailerEnvironment holds per-environment mailer overrides.
type MailerEnvironment struct {
	Environment string `hcl:"env,label"`
	Renderer    string `hcl:"renderer,optional"`
	Transport   string `hcl:"transport,optional"`
	From        string `hcl:"from,optional"`
	Host        string `hcl:"host,optional"`
	Port        string `hcl:"port,optional"`
	Username    string `hcl:"username,optional"`
	Password    string `hcl:"password,optional"`
	TLS         string `hcl:"tls,optional"`
}

// Mailer holds mailer configuration with optional per-environment overrides.
type Mailer struct {
	Renderer  string             `hcl:"renderer,optional"`
	Transport string             `hcl:"transport,optional"`
	From      string             `hcl:"from,optional"`
	Host      string             `hcl:"host,optional"`
	Port      string             `hcl:"port,optional"`
	Username  string             `hcl:"username,optional"`
	Password  string             `hcl:"password,optional"`
	TLS       string             `hcl:"tls,optional"`
	Envs      []MailerEnvironment `hcl:"env,block"`
}

// MailerForEnv merges the base mailer config with env-specific overrides.
func (pf *Polafile) MailerForEnv(env string) Mailer {
	if pf.Mailer == nil {
		return Mailer{}
	}
	base := Mailer{
		Renderer:  pf.Mailer.Renderer,
		Transport: pf.Mailer.Transport,
		From:      pf.Mailer.From,
		Host:      pf.Mailer.Host,
		Port:      pf.Mailer.Port,
		Username:  pf.Mailer.Username,
		Password:  pf.Mailer.Password,
		TLS:       pf.Mailer.TLS,
	}
	for _, e := range pf.Mailer.Envs {
		if e.Environment == env {
			override := Mailer{
				Renderer:  e.Renderer,
				Transport: e.Transport,
				From:      e.From,
				Host:      e.Host,
				Port:      e.Port,
				Username:  e.Username,
				Password:  e.Password,
				TLS:       e.TLS,
			}
			_ = mergo.Merge(&base, &override, mergo.WithOverride)
			break
		}
	}
	return base
}

// MailerTransport returns the configured mailer transport for the given environment,
// falling back to base config, then "log".
func (pf *Polafile) MailerTransport(env string) string {
	merged := pf.MailerForEnv(env)
	if merged.Transport != "" {
		return merged.Transport
	}
	return "log"
}

// MailerFrom returns the configured default from address for the given environment.
func (pf *Polafile) MailerFrom(env string) string {
	merged := pf.MailerForEnv(env)
	return merged.From
}

// MailerRenderer returns the configured mailer renderer for the given environment,
// falling back to base config, then "react".
func (pf *Polafile) MailerRenderer(env string) string {
	merged := pf.MailerForEnv(env)
	if merged.Renderer != "" {
		return merged.Renderer
	}
	return "react"
}

// ---------- ImageProcessing ----------

// ImageProcessingEnvironment holds per-environment image processing overrides.
type ImageProcessingEnvironment struct {
	Environment string `hcl:"env,label"`
	Enabled     *bool  `hcl:"enabled,optional"`
	Adapter     string `hcl:"adapter,optional"`
	Path        string `hcl:"path,optional"`
	MaxWidth    int    `hcl:"max_width,optional"`
	MaxHeight   int    `hcl:"max_height,optional"`
	Format      string `hcl:"format,optional"`
}

// ImageProcessing holds image processing configuration with optional per-environment overrides.
type ImageProcessing struct {
	Enabled   bool                         `hcl:"enabled,optional"`
	Adapter   string                       `hcl:"adapter,optional"`
	Path      string                       `hcl:"path,optional"`
	MaxWidth  int                          `hcl:"max_width,optional"`
	MaxHeight int                          `hcl:"max_height,optional"`
	Format    string                       `hcl:"format,optional"`
	Envs      []ImageProcessingEnvironment `hcl:"env,block"`
}

// ImageProcessingEnabled returns whether image processing is enabled for the given environment.
// Resolution: env override > base > default (false).
func (pf *Polafile) ImageProcessingEnabled(env string) bool {
	if pf.ImageProcessing == nil {
		return false
	}
	base := pf.ImageProcessing.Enabled
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.Enabled != nil {
			return *e.Enabled
		}
	}
	return base
}

// ImageProcessingAdapter returns the configured image processing adapter for the given environment,
// falling back to base config, then "imaging".
func (pf *Polafile) ImageProcessingAdapter(env string) string {
	if pf.ImageProcessing == nil {
		return ""
	}
	if !pf.ImageProcessingEnabled(env) {
		return ""
	}
	base := pf.ImageProcessing.Adapter
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.Adapter != "" {
			return e.Adapter
		}
	}
	if base != "" {
		return base
	}
	return "imaging"
}

// ImageProcessingPath returns the configured image processing path prefix,
// falling back to base config, then "/_image".
func (pf *Polafile) ImageProcessingPath(env string) string {
	if pf.ImageProcessing == nil {
		return "/_image"
	}
	base := pf.ImageProcessing.Path
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.Path != "" {
			return e.Path
		}
	}
	if base != "" {
		return base
	}
	return "/_image"
}

// ImageProcessingMaxWidth returns the configured max width,
// falling back to base config, then 4096.
func (pf *Polafile) ImageProcessingMaxWidth(env string) int {
	if pf.ImageProcessing == nil {
		return 4096
	}
	base := pf.ImageProcessing.MaxWidth
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.MaxWidth > 0 {
			return e.MaxWidth
		}
	}
	if base > 0 {
		return base
	}
	return 4096
}

// ImageProcessingMaxHeight returns the configured max height,
// falling back to base config, then 4096.
func (pf *Polafile) ImageProcessingMaxHeight(env string) int {
	if pf.ImageProcessing == nil {
		return 4096
	}
	base := pf.ImageProcessing.MaxHeight
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.MaxHeight > 0 {
			return e.MaxHeight
		}
	}
	if base > 0 {
		return base
	}
	return 4096
}

// ImageProcessingFormat returns the configured default output format,
// falling back to base config, then "jpeg".
func (pf *Polafile) ImageProcessingFormat(env string) string {
	if pf.ImageProcessing == nil {
		return "jpeg"
	}
	base := pf.ImageProcessing.Format
	for _, e := range pf.ImageProcessing.Envs {
		if e.Environment == env && e.Format != "" {
			return e.Format
		}
	}
	if base != "" {
		return base
	}
	return "jpeg"
}

// ---------- MCP ----------

// MCPEnvironment holds per-environment MCP overrides.
type MCPEnvironment struct {
	Environment  string `hcl:"env,label"`
	Enabled      *bool  `hcl:"enabled,optional"`
	Transport    string `hcl:"transport,optional"`
	Mount        string `hcl:"mount,optional"`
	Name         string `hcl:"name,optional"`
	Version      string `hcl:"version,optional"`
	Instructions string `hcl:"instructions,optional"`
}

// MCP holds Model Context Protocol server configuration with optional
// per-environment overrides. Transport is "http" (default), "sse", or "stdio".
type MCP struct {
	Enabled      bool             `hcl:"enabled,optional"`
	Transport    string           `hcl:"transport,optional"`
	Mount        string           `hcl:"mount,optional"`
	Name         string           `hcl:"name,optional"`
	Version      string           `hcl:"version,optional"`
	Instructions string           `hcl:"instructions,optional"`
	Envs         []MCPEnvironment `hcl:"env,block"`
}

// MCPEnabled returns whether the MCP server is enabled for the given environment.
// Resolution: env override > base > default (false).
func (pf *Polafile) MCPEnabled(env string) bool {
	if pf.MCP == nil {
		return false
	}
	for _, e := range pf.MCP.Envs {
		if e.Environment == env && e.Enabled != nil {
			return *e.Enabled
		}
	}
	return pf.MCP.Enabled
}

// MCPForEnv merges the base MCP config with env-specific overrides.
func (pf *Polafile) MCPForEnv(env string) MCP {
	if pf.MCP == nil {
		return MCP{}
	}
	base := MCP{
		Transport:    pf.MCP.Transport,
		Mount:        pf.MCP.Mount,
		Name:         pf.MCP.Name,
		Version:      pf.MCP.Version,
		Instructions: pf.MCP.Instructions,
	}
	for _, e := range pf.MCP.Envs {
		if e.Environment == env {
			override := MCP{
				Transport:    e.Transport,
				Mount:        e.Mount,
				Name:         e.Name,
				Version:      e.Version,
				Instructions: e.Instructions,
			}
			_ = mergo.Merge(&base, &override, mergo.WithOverride)
			break
		}
	}
	return base
}

// MCPTransport returns the configured MCP transport for the given environment,
// defaulting to "http".
func (pf *Polafile) MCPTransport(env string) string {
	merged := pf.MCPForEnv(env)
	if merged.Transport != "" {
		return merged.Transport
	}
	return "http"
}

// MCPMount returns the configured MCP mount path, defaulting to "/mcp".
func (pf *Polafile) MCPMount(env string) string {
	merged := pf.MCPForEnv(env)
	if merged.Mount != "" {
		return merged.Mount
	}
	return "/mcp"
}

// MCPName returns the configured MCP server name, falling back to "pola-mcp".
func (pf *Polafile) MCPName(env string) string {
	merged := pf.MCPForEnv(env)
	if merged.Name != "" {
		return merged.Name
	}
	return "pola-mcp"
}

// MCPVersion returns the configured MCP server version, falling back to "0.1.0".
func (pf *Polafile) MCPVersion(env string) string {
	merged := pf.MCPForEnv(env)
	if merged.Version != "" {
		return merged.Version
	}
	return "0.1.0"
}

// MCPInstructions returns the configured MCP instructions string.
func (pf *Polafile) MCPInstructions(env string) string {
	return pf.MCPForEnv(env).Instructions
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

	// Storage block.
	if pf.Storage != nil {
		blockBody.AppendNewline()
		sBlock := blockBody.AppendNewBlock("storage", nil)
		sBody := sBlock.Body()
		setAttr(sBody, "driver", pf.Storage.Driver)
		setAttr(sBody, "root", pf.Storage.Root)
		setAttr(sBody, "config_path", pf.Storage.ConfigPath)
		for _, e := range pf.Storage.Envs {
			envBlock := sBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			setAttr(envBody, "driver", e.Driver)
			setAttr(envBody, "root", e.Root)
			setAttr(envBody, "config_path", e.ConfigPath)
		}
	}

	// Mailer block.
	if pf.Mailer != nil {
		blockBody.AppendNewline()
		mBlock := blockBody.AppendNewBlock("mailer", nil)
		mBody := mBlock.Body()
		setAttr(mBody, "renderer", pf.Mailer.Renderer)
		setAttr(mBody, "transport", pf.Mailer.Transport)
		setAttr(mBody, "from", pf.Mailer.From)
		setAttr(mBody, "host", pf.Mailer.Host)
		setAttr(mBody, "port", pf.Mailer.Port)
		setAttr(mBody, "username", pf.Mailer.Username)
		setAttr(mBody, "password", pf.Mailer.Password)
		setAttr(mBody, "tls", pf.Mailer.TLS)
		for _, e := range pf.Mailer.Envs {
			envBlock := mBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			setAttr(envBody, "renderer", e.Renderer)
			setAttr(envBody, "transport", e.Transport)
			setAttr(envBody, "from", e.From)
			setAttr(envBody, "host", e.Host)
			setAttr(envBody, "port", e.Port)
			setAttr(envBody, "username", e.Username)
			setAttr(envBody, "password", e.Password)
			setAttr(envBody, "tls", e.TLS)
		}
	}

	// ImageProcessing block.
	if pf.ImageProcessing != nil {
		blockBody.AppendNewline()
		ipBlock := blockBody.AppendNewBlock("image_processing", nil)
		ipBody := ipBlock.Body()
		ipBody.SetAttributeValue("enabled", cty.BoolVal(pf.ImageProcessing.Enabled))
		setAttr(ipBody, "adapter", pf.ImageProcessing.Adapter)
		setAttr(ipBody, "path", pf.ImageProcessing.Path)
		if pf.ImageProcessing.MaxWidth > 0 {
			ipBody.SetAttributeValue("max_width", cty.NumberIntVal(int64(pf.ImageProcessing.MaxWidth)))
		}
		if pf.ImageProcessing.MaxHeight > 0 {
			ipBody.SetAttributeValue("max_height", cty.NumberIntVal(int64(pf.ImageProcessing.MaxHeight)))
		}
		setAttr(ipBody, "format", pf.ImageProcessing.Format)
		for _, e := range pf.ImageProcessing.Envs {
			envBlock := ipBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			setAttr(envBody, "adapter", e.Adapter)
			setAttr(envBody, "path", e.Path)
			if e.MaxWidth > 0 {
				envBody.SetAttributeValue("max_width", cty.NumberIntVal(int64(e.MaxWidth)))
			}
			if e.MaxHeight > 0 {
				envBody.SetAttributeValue("max_height", cty.NumberIntVal(int64(e.MaxHeight)))
			}
			setAttr(envBody, "format", e.Format)
		}
	}

	// MCP block.
	if pf.MCP != nil {
		blockBody.AppendNewline()
		mcpBlock := blockBody.AppendNewBlock("mcp", nil)
		mcpBody := mcpBlock.Body()
		mcpBody.SetAttributeValue("enabled", cty.BoolVal(pf.MCP.Enabled))
		setAttr(mcpBody, "transport", pf.MCP.Transport)
		setAttr(mcpBody, "mount", pf.MCP.Mount)
		setAttr(mcpBody, "name", pf.MCP.Name)
		setAttr(mcpBody, "version", pf.MCP.Version)
		setAttr(mcpBody, "instructions", pf.MCP.Instructions)
		for _, e := range pf.MCP.Envs {
			envBlock := mcpBody.AppendNewBlock("env", []string{e.Environment})
			envBody := envBlock.Body()
			if e.Enabled != nil {
				envBody.SetAttributeValue("enabled", cty.BoolVal(*e.Enabled))
			}
			setAttr(envBody, "transport", e.Transport)
			setAttr(envBody, "mount", e.Mount)
			setAttr(envBody, "name", e.Name)
			setAttr(envBody, "version", e.Version)
			setAttr(envBody, "instructions", e.Instructions)
		}
	}

	// Testing block.
	if pf.Testing != nil {
		blockBody.AppendNewline()
		tBlock := blockBody.AppendNewBlock("testing", nil)
		tBody := tBlock.Body()
		if pf.Testing.GenerateTests != nil {
			tBody.SetAttributeValue("generate_tests", cty.BoolVal(*pf.Testing.GenerateTests))
		}
		setAttr(tBody, "framework", pf.Testing.Framework)
	}

	path := filepath.Join(dir, Filename)
	if err := os.WriteFile(path, f.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", Filename, err)
	}
	return nil
}
