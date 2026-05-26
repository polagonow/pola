// Package autoload defines the pluggable overlay interface and registry
// for the Pola CLI. Each autoload is a self-contained package that registers
// itself via init().
package autoload

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/polafile"
)

// Autoload is a pluggable overlay contributor for the Pola CLI.
// Each implementation generates overlay files (Go source injected via
// the -overlay flag) and populates shared Discovery state for downstream
// autoloads that depend on earlier discovery results.
type Autoload interface {
	// Name returns the autoload identifier (e.g. "actionbridge", "routes").
	Name() string
	// Priority determines execution order — lower values run first.
	// Use multiples of 100 so future autoloads can slot in between.
	Priority() int
	// Contribute generates overlay files and populates ctx.Discovery.
	// Generated files are added to ctx.Replace (virtual path → real path).
	Contribute(ctx *Context) error
}

// Context holds shared state passed to every Autoload.Contribute call.
type Context struct {
	ProjectDir string
	TmpDir     string
	ModPath    string
	Opts       PluginOpts
	Replace    map[string]string // overlay JSON entries: virtual → real
	Discovery  *Discovery        // shared accumulator across autoloads
	Verbose    bool
}

// Discovery accumulates results from earlier autoloads so that later ones
// (e.g. pluginimports) can read them without coupling to specific autoloads.
type Discovery struct {
	ActionsImport string
	HasActions    bool
	RoutePkgs     []RoutePackageInfo
	RepoDisco     *RepoDiscovery
	SvcDisco      *SvcDiscovery
	TSOutPath     string
}

// Result holds the output from Run.
type Result struct {
	OverlayPath string
	TmpDir      string
	TSOutPath   string
}

// PluginOpts holds parameters for plugin generation.
type PluginOpts struct {
	PolaPackage     string
	Engine          string
	Bundler         string
	Renderer        string
	Router          string
	CSS             string
	Cache           string
	Database        string // ORM name: "ent", "gorm", "beego", or "" for none.
	DatabaseAdapter string // "postgresql", "mysql", "sqlite"
	DatabaseURL     string
	DatabaseHost    string
	DatabasePort    string
	DatabaseUser    string
	DatabasePass    string
	DatabaseName    string
	CSRF            bool
	SecurityHeaders bool
	Dev             bool
	Embed           bool
	AppDir          string // path to the web app directory (default: web)
	ActionsDir      string // path to actions directory (default: ./actions)
	TSOut           string // path to generated .d.ts file
	StorageDriver   string // "fs", "rclone", or "" for none
	StorageRoot     string // local dir for fs, "remote:path" for rclone
	StorageConfig   string // optional rclone config path
	MailerRenderer  string
	MailerTransport string
	MailerFrom      string
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLS         bool
	ImageProcessing string
}

// RoutePackageInfo holds metadata about a discovered route package.
type RoutePackageInfo struct {
	ImportPath string // e.g. "test-app/routes/kampala/uganda"
	AbsDir     string // e.g. "/abs/path/routes/kampala/uganda"
	PkgName    string // e.g. "uganda"
}

// PluginEntry holds a discovered plugin name with its pre-computed snake_case form.
type PluginEntry struct {
	Name      string // e.g. "User"
	SnakeName string // e.g. "user"
}

// RepoDiscovery holds discovered repository registration info for the overlay.
type RepoDiscovery struct {
	ImportPath   string        // e.g. "myapp/repositories/gorm"
	RepoImport   string        // e.g. "myapp/repositories"
	ModulePath   string        // e.g. "myapp"
	EntClientDir string        // e.g. "db/orm/ent"
	RepoDir      string        // e.g. "repositories"
	ORM          string        // e.g. "gorm", "ent", "beego"
	PkgName      string        // e.g. "gorm"
	Repositories []PluginEntry // e.g. [{Name: "User", SnakeName: "user"}]
}

// SvcDiscovery holds discovered service constructor info for the overlay.
type SvcDiscovery struct {
	ImportPath string        // e.g. "myapp/services"
	RepoImport string        // e.g. "myapp/repositories"
	PkgName    string        // e.g. "services"
	Services   []PluginEntry // e.g. [{Name: "User", SnakeName: "user"}]
}

// RouteServiceDep holds the service dependency info discovered in a route package.
type RouteServiceDep struct {
	ServicePkg  string // e.g. "services"
	ServiceType string // e.g. "PostService"
	ServicePath string // e.g. "myapp/services"
	HasStorage  bool   // true if Route struct has a storage.Storage field
	BlobRepo    string // e.g. "StorageBlobRepository" if Route depends on a *Repository field; empty if none
}

// EnvOrBool returns the boolean value from an env var if set, otherwise the fallback.
func EnvOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1"
}

// DatabaseORM returns the configured ORM name if a database block exists, or "" otherwise.
func DatabaseORM(pf *polafile.Polafile) string {
	if pf == nil || pf.Database == nil {
		return ""
	}
	return pf.DatabaseORM()
}

// PopulateStorageOpts fills storage-related fields in PluginOpts from the Polafile.
func PopulateStorageOpts(opts *PluginOpts, pf *polafile.Polafile, env string) {
	if pf == nil || pf.Storage == nil {
		return
	}
	merged := pf.StorageForEnv(env)
	opts.StorageDriver = merged.Driver
	opts.StorageRoot = merged.Root
	opts.StorageConfig = merged.ConfigPath
}

// PopulateDatabaseOpts fills database-related fields in PluginOpts from the Polafile.
func PopulateDatabaseOpts(opts *PluginOpts, pf *polafile.Polafile, env string) {
	opts.Database = DatabaseORM(pf)
	if opts.Database == "" {
		return
	}
	merged := pf.DatabaseForEnv(env)
	opts.DatabaseAdapter = pf.DatabaseAdapter(env)
	opts.DatabaseURL = merged.URL
	opts.DatabaseHost = merged.Host
	opts.DatabasePort = merged.Port
	opts.DatabaseUser = merged.User
	opts.DatabasePass = merged.Password
	opts.DatabaseName = merged.Name
}

// ApplyMailerOpts populates mailer-related fields in PluginOpts from the Polafile.
func ApplyMailerOpts(opts *PluginOpts, pf *polafile.Polafile, env string) {
	if pf == nil || pf.Mailer == nil {
		return
	}
	merged := pf.MailerForEnv(env)
	opts.MailerRenderer = cmp.Or(merged.Renderer, "react")
	opts.MailerTransport = cmp.Or(merged.Transport, "log")
	opts.MailerFrom = cmp.Or(merged.From, os.Getenv("MAILER_FROM"))
	opts.SMTPHost = cmp.Or(merged.Host, os.Getenv("SMTP_HOST"))
	opts.SMTPPort = cmp.Or(merged.Port, os.Getenv("SMTP_PORT"))
	opts.SMTPUsername = cmp.Or(merged.Username, os.Getenv("SMTP_USERNAME"))
	opts.SMTPPassword = cmp.Or(merged.Password, os.Getenv("SMTP_PASSWORD"))
	tlsVal := cmp.Or(merged.TLS, os.Getenv("SMTP_TLS"))
	opts.SMTPTLS = tlsVal == "true" || tlsVal == "starttls"
}

// ReadModulePath reads the module path from go.mod in the given directory.
func ReadModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// CondStr returns a if cond is true, b otherwise.
func CondStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
