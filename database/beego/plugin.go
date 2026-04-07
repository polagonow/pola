// Package beego provides a Pola plugin that registers a Beego ORM Ormer in the DI container.
package beego

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/database/dsn"
)

// drivers is the adapter registry. Adapter sub-packages register themselves via init().
var drivers = map[string]string{}

// RegisterDriver registers a Beego ORM driver name for the given adapter.
func RegisterDriver(adapter, driverName string) {
	drivers[adapter] = driverName
}

// Option configures the Beego database plugin.
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

type beegoPlugin struct {
	cfg      config
	registry *core.Registry
}

// Plugin returns a Pola plugin that registers orm.Ormer in the DI container.
// Options set explicit values; unset fields fall back to DATABASE_* env vars.
func Plugin(opts ...Option) core.Plugin {
	p := &beegoPlugin{}
	for _, o := range opts {
		o(&p.cfg)
	}
	return p
}

func (p *beegoPlugin) Name() string { return "beego-database" }

func (p *beegoPlugin) Register(r *core.Registry) {
	p.registry = r
	core.Provide[orm.Ormer](r, func() (orm.Ormer, error) {
		c := p.cfg.Config.WithEnvFallback()
		connStr := dsn.Build(c)
		driverName, ok := drivers[c.Adapter]
		if !ok {
			return nil, fmt.Errorf("beego: no driver registered for adapter %q (import the adapter sub-package)", c.Adapter)
		}

		if err := orm.RegisterDataBase("default", driverName, connStr); err != nil {
			return nil, fmt.Errorf("beego: register database: %w", err)
		}
		return orm.NewOrm(), nil
	})
}

func (p *beegoPlugin) Start(_ context.Context) error {
	_, err := core.Invoke[orm.Ormer](p.registry)
	return err
}

func (p *beegoPlugin) Shutdown(_ context.Context) error {
	// Beego manages its own connection pool internally.
	return nil
}

