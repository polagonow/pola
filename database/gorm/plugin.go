// Package gorm provides a Pola plugin that registers a *gorm.DB connection in the DI container.
package gorm

import (
	"context"
	"fmt"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/database/dsn"
	gormpkg "gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
)

// Option configures the GORM database plugin.
type Option func(*config)

type config struct {
	dsn.Config
}

// WithURL sets the full connection string.
func WithURL(url string) Option { return func(c *config) { c.URL = url } }

// WithHost sets the database host.
func WithHost(host string) Option { return func(c *config) { c.Host = host } }

// WithPort sets the database port.
func WithPort(port string) Option { return func(c *config) { c.Port = port } }

// WithUser sets the database user.
func WithUser(user string) Option { return func(c *config) { c.User = user } }

// WithPassword sets the database password.
func WithPassword(password string) Option { return func(c *config) { c.Password = password } }

// WithName sets the database name.
func WithName(name string) Option { return func(c *config) { c.Name = name } }

// WithAdapter sets the database adapter (postgresql, mysql, sqlite).
func WithAdapter(adapter string) Option { return func(c *config) { c.Adapter = adapter } }

type gormPlugin struct {
	cfg config
	db  *gormpkg.DB
}

// Plugin returns a Pola plugin that registers *gorm.DB in the DI container.
// Options set explicit values; any unset fields fall back to DATABASE_* env vars.
func Plugin(opts ...Option) core.Plugin {
	p := &gormPlugin{}
	for _, o := range opts {
		o(&p.cfg)
	}
	return p
}

func (p *gormPlugin) Name() string { return "gorm-database" }

func (p *gormPlugin) Register(r *core.Registry) {
	core.Provide[*gormpkg.DB](r, func() (*gormpkg.DB, error) {
		c := p.cfg.Config.WithEnvFallback()
		connStr := dsn.Build(c)

		dialector, err := dialectorFor(c.Adapter, connStr)
		if err != nil {
			return nil, err
		}

		db, err := gormpkg.Open(dialector, &gormpkg.Config{})
		if err != nil {
			return nil, fmt.Errorf("gorm open: %w", err)
		}
		p.db = db
		return db, nil
	})
}

func (p *gormPlugin) Shutdown(_ context.Context) error {
	if p.db != nil {
		sqlDB, err := p.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func dialectorFor(adapter, connStr string) (gormpkg.Dialector, error) {
	switch adapter {
	case "postgresql", "postgres":
		return postgres.Open(connStr), nil
	case "mysql":
		return mysql.Open(connStr), nil
	case "sqlite":
		return sqlite.Open(connStr), nil
	default:
		return nil, fmt.Errorf("gorm: unsupported adapter %q", adapter)
	}
}
