package xorm

import (
	"fmt"
	"time"

	"github.com/gorilla/sessions"
	"github.com/lafriks/xormstore"
	"xorm.io/xorm"
)

type Option func(*config)

type config struct {
	dsn             string
	driver          string
	tableName       string
	cleanupInterval time.Duration
}

func WithDSN(dsn string) Option {
	return func(c *config) { c.dsn = dsn }
}

func WithDriver(driver string) Option {
	return func(c *config) { c.driver = driver }
}

func WithTableName(name string) Option {
	return func(c *config) { c.tableName = name }
}

func WithCleanupInterval(d time.Duration) Option {
	return func(c *config) { c.cleanupInterval = d }
}

func NewStore(keyPairs []byte, opts ...Option) (sessions.Store, error) {
	cfg := &config{
		driver:          "sqlite3",
		tableName:       "http_sessions",
		cleanupInterval: 5 * time.Minute,
	}
	for _, o := range opts {
		o(cfg)
	}

	engine, err := xorm.NewEngine(cfg.driver, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("session/xorm: engine: %w", err)
	}

	store, err := xormstore.NewOptions(engine, xormstore.Options{
		TableName: cfg.tableName,
	}, keyPairs)
	if err != nil {
		return nil, fmt.Errorf("session/xorm: create store: %w", err)
	}

	go store.PeriodicCleanup(cfg.cleanupInterval, nil)

	return store, nil
}

func MustNew(keyPairs []byte, opts ...Option) sessions.Store {
	s, err := NewStore(keyPairs, opts...)
	if err != nil {
		panic(err)
	}
	return s
}
