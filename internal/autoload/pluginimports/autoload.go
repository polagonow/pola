// Package pluginimports implements the plugin imports overlay autoload.
// It generates pola_plugins.go containing explicit Plugin() calls.
package pluginimports

import (
	"cmp"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/plugins_go.tmpl
var templates embed.FS

var pluginsTmpl = template.Must(
	template.New("plugins_go.tmpl").ParseFS(templates, "_templates/plugins_go.tmpl"),
)

type autoloadImpl struct{}

func init() {
	autoload.Register(&autoloadImpl{})
}

func (a *autoloadImpl) Name() string { return "pluginimports" }
func (a *autoloadImpl) Priority() int { return 900 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	routeImports := make([]string, len(ctx.Discovery.RoutePkgs))
	for i, rp := range ctx.Discovery.RoutePkgs {
		routeImports[i] = rp.ImportPath
	}

	pluginsSrc, err := GenerateSource(ctx.Opts, ctx.Discovery.ActionsImport, routeImports, ctx.Discovery.RepoDisco, ctx.Discovery.EntClientDisco, ctx.Discovery.SvcDisco, ctx.Discovery.MCPDisco)
	if err != nil {
		return fmt.Errorf("generate plugins: %w", err)
	}

	pluginsPath := filepath.Join(ctx.TmpDir, "pola_plugins.go")
	if err := os.WriteFile(pluginsPath, pluginsSrc, 0o644); err != nil {
		return fmt.Errorf("write plugins: %w", err)
	}

	absProjectDir, err := filepath.Abs(ctx.ProjectDir)
	if err != nil {
		return fmt.Errorf("abs project dir: %w", err)
	}
	ctx.Replace[filepath.Join(absProjectDir, "pola_plugins.go")] = pluginsPath

	return nil
}

// GenerateSource returns the source for pola_plugins.go. This is exported
// so that the "pola new" command can call it directly (without running the
// full autoload pipeline) to produce a temporary file for go mod tidy.
func GenerateSource(opts autoload.PluginOpts, actionsImport string, routeImports []string, repoDisco *autoload.RepoDiscovery, entClientDisco *autoload.EntClientDiscovery, svcDisco *autoload.SvcDiscovery, mcpDisco *autoload.MCPDiscovery) ([]byte, error) {
	hasCSS := opts.CSS != "" && opts.CSS != "none"
	hasCache := opts.Cache != "" && opts.Cache != "none"
	hasDatabase := opts.Database != ""
	hasImageProcessing := opts.ImageProcessing != "" && opts.ImageProcessing != "none"
	hasCSRF := opts.CSRF
	hasSecurityHeaders := opts.SecurityHeaders
	hasStorage := opts.StorageDriver != ""
	hasMailer := opts.MailerRenderer != "" || opts.MailerTransport != ""
	hasRateLimit := opts.RateLimit
	hasSession := opts.Session
	hasFlash := opts.Flash
	hasI18n := opts.I18n

	var buf strings.Builder
	err := pluginsTmpl.Execute(&buf, struct {
		PolaPackage     string
		Framework       string
		Engine          string
		Bundler         string
		Renderer        string
		Router          string
		CSS             string
		Cache           string
		Database        string
		DatabaseAdapter string
		DatabaseURL     string
		DatabaseHost    string
		DatabasePort    string
		DatabaseUser    string
		DatabasePass    string
		DatabaseName    string
		CSRF            bool
		SecurityHeaders bool
		ImageProcessing string
		Dev             bool
		Embed           bool
		HasRoutes       bool
		ActionsImport   string
		RouteImports    []string
		RepoPlugins     *autoload.RepoDiscovery
		EntClient       *autoload.EntClientDiscovery
		ServicePlugins  *autoload.SvcDiscovery
		HasStorage      bool
		StorageDriver   string // "fs" or "rclone"
		StorageRoot     string // local dir for fs, "remote:path" for rclone
		StorageConfig   string // optional rclone config path
		HasMailer       bool
		MailerRenderer  string
		MailerTransport string
		MailerFrom      string
		SMTPHost        string
		SMTPPort        string
		SMTPUsername    string
		SMTPPassword    string
		SMTPTLS         bool

		HasMCP          bool
		MCPTransport    string
		MCPMount        string
		MCPName         string
		MCPVersion      string
		MCPInstructions string
		MCPDisco        *autoload.MCPDiscovery

		RateLimit      bool
		RateLimitRPS   float64
		RateLimitBurst int
		Session         bool
		SessionStore    string
		SessionHost     string
		SessionPort     string
		SessionPassword string
		SessionDB       string
		SessionDSN      string
		Flash           bool
		I18n           bool
		I18nLocale     string
		I18nDirectory  string

		APIOnly bool
	}{
		PolaPackage:     opts.PolaPackage,
		Framework:       cmp.Or(opts.Framework, "std"),
		Engine:          opts.Engine,
		Bundler:         opts.Bundler,
		Renderer:        opts.Renderer,
		Router:          opts.Router,
		CSS:             autoload.CondStr(hasCSS, opts.CSS, ""),
		Cache:           autoload.CondStr(hasCache, opts.Cache, ""),
		Database:        autoload.CondStr(hasDatabase, opts.Database, ""),
		DatabaseAdapter: opts.DatabaseAdapter,
		DatabaseURL:     opts.DatabaseURL,
		DatabaseHost:    opts.DatabaseHost,
		DatabasePort:    opts.DatabasePort,
		DatabaseUser:    opts.DatabaseUser,
		DatabasePass:    opts.DatabasePass,
		DatabaseName:    opts.DatabaseName,
		CSRF:            hasCSRF,
		SecurityHeaders: hasSecurityHeaders,
		ImageProcessing: autoload.CondStr(hasImageProcessing, opts.ImageProcessing, ""),
		Dev:             opts.Dev,
		Embed:           opts.Embed,
		HasRoutes:       len(routeImports) > 0,
		ActionsImport:   actionsImport,
		RouteImports:    routeImports,
		RepoPlugins:     repoDisco,
		EntClient:       entClientDisco,
		ServicePlugins:  svcDisco,
		HasStorage:      hasStorage,
		StorageDriver:   opts.StorageDriver,
		StorageRoot:     opts.StorageRoot,
		StorageConfig:   opts.StorageConfig,
		HasMailer:       hasMailer,
		MailerRenderer:  cmp.Or(opts.MailerRenderer, "react"),
		MailerTransport: cmp.Or(opts.MailerTransport, "log"),
		MailerFrom:      opts.MailerFrom,
		SMTPHost:        opts.SMTPHost,
		SMTPPort:        opts.SMTPPort,
		SMTPUsername:    opts.SMTPUsername,
		SMTPPassword:    opts.SMTPPassword,
		SMTPTLS:         opts.SMTPTLS,

		HasMCP:          opts.HasMCP,
		MCPTransport:    cmp.Or(opts.MCPTransport, "http"),
		MCPMount:        cmp.Or(opts.MCPMount, "/mcp"),
		MCPName:         opts.MCPName,
		MCPVersion:      opts.MCPVersion,
		MCPInstructions: opts.MCPInstructions,
		MCPDisco:        mcpDisco,

		RateLimit:      hasRateLimit,
		RateLimitRPS:   opts.RateLimitRPS,
		RateLimitBurst: opts.RateLimitBurst,
		Session:         hasSession,
		SessionStore:    cmp.Or(opts.SessionStore, "cookie"),
		SessionHost:     opts.SessionHost,
		SessionPort:     opts.SessionPort,
		SessionPassword: opts.SessionPassword,
		SessionDB:       opts.SessionDB,
		SessionDSN:      opts.SessionDSN,
		Flash:           hasFlash,
		I18n:           hasI18n,
		I18nLocale:     cmp.Or(opts.I18nLocale, "en"),
		I18nDirectory:  cmp.Or(opts.I18nDirectory, "locales"),

		APIOnly: opts.APIOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("execute plugins template: %w", err)
	}
	return []byte(buf.String()), nil
}
