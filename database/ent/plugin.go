// Package ent provides a Pola plugin that registers an Ent SQL driver in the DI container.
//
// Since *ent.Client is codegen'd per project, this plugin registers *entsql.Driver
// and *sql.DB. Users create their project-specific client via:
//
//	drv := core.MustInvoke[*entsql.Driver](registry)
//	client := ent.NewClient(ent.Driver(drv))
package ent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/database/dsn"

	entsql "entgo.io/ent/dialect/sql"
)

// Driver describes a database/sql driver and Ent dialect for one adapter.
// Adapter sub-packages export a Driver() constructor, e.g.
// database/ent/postgresql.Driver().
type Driver struct {
	Name       string // adapter name, e.g. "postgresql"
	SQLDriver  string // database/sql driver name, e.g. "postgres"
	EntDialect string // e.g. "postgres", "mysql", "sqlite3"
}

// Option configures the Ent database plugin.
type Option func(*config)

type config struct {
	dsn.Config
	drivers []Driver
}

// WithDriver makes a driver selectable by the configured adapter name.
// May be passed multiple times (e.g. sqlite for dev, postgresql for prod,
// selected at runtime via WithAdapter / POLA_DATABASE_ADAPTER).
func WithDriver(d Driver) Option { return func(c *config) { c.drivers = append(c.drivers, d) } }

func (c *config) driverFor(adapter string) (Driver, error) {
	if adapter == "" && len(c.drivers) == 1 {
		return c.drivers[0], nil
	}
	for _, d := range c.drivers {
		if d.Name == adapter {
			return d, nil
		}
	}
	return Driver{}, fmt.Errorf(
		"ent: no driver provided for adapter %q (pass databaseent.WithDriver(<adapter>.Driver()))", adapter)
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

type entPlugin struct {
	cfg      config
	db       *sql.DB
	registry *core.Registry
}

// Plugin returns a Pola plugin that registers *entsql.Driver and *sql.DB
// in the DI container. Options set explicit values; unset fields fall back
// to DATABASE_* env vars.
func Plugin(opts ...Option) core.Plugin {
	p := &entPlugin{}
	for _, o := range opts {
		o(&p.cfg)
	}
	return p
}

func (p *entPlugin) Name() string { return "ent-database" }

func (p *entPlugin) Register(r *core.Registry) {
	p.registry = r
	core.Provide[*entsql.Driver](r, func() (*entsql.Driver, error) {
		c := p.cfg.Config.WithEnvFallback()
		connStr := dsn.Build(c)
		info, err := p.cfg.driverFor(c.Adapter)
		if err != nil {
			return nil, err
		}

		db, err := sql.Open(info.SQLDriver, connStr)
		if err != nil {
			return nil, fmt.Errorf("ent: sql.Open: %w", err)
		}
		if err := db.Ping(); err != nil {
			db.Close()
			return nil, fmt.Errorf("ent: ping: %w", err)
		}
		p.db = db

		drv := entsql.OpenDB(info.EntDialect, db)
		return drv, nil
	})

	core.Provide[*sql.DB](r, func() (*sql.DB, error) {
		if p.db == nil {
			// Trigger driver creation to ensure db is opened.
			if _, err := core.Invoke[*entsql.Driver](r); err != nil {
				return nil, err
			}
		}
		return p.db, nil
	})
}

func (p *entPlugin) Start(_ context.Context) error {
	_, err := core.Invoke[*entsql.Driver](p.registry)
	return err
}

func (p *entPlugin) Shutdown(_ context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
